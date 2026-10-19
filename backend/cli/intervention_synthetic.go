package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const syntheticChannelExternalID = "ccma-intervention-operator-demo"
const syntheticJobDescription = "ccma.synthetic.intervention.v1:operator-provisioned"
const syntheticWaitExternalID = "ccma-synthetic-wait-data"
const syntheticCompletedMessageID = "ccma-synthetic-wait-data-complete"
const syntheticCompletedMessageText = "Tôi khiếu nại vì đơn hàng bị chậm và đang chờ phản hồi."

type SyntheticDemoIDs struct {
	TenantID              string `json:"tenant_id"`
	ChannelID             string `json:"channel_id"`
	JobID                 string `json:"job_id"`
	ComplaintConversation string `json:"complaint_conversation_id"`
	WaitConversation      string `json:"wait_conversation_id"`
}

func interventionOperator(tx *gorm.DB, tenantID, issuerID string) error {
	var tenant models.Tenant
	if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", tenantID).Take(&tenant).Error != nil {
		return errors.New("tenant_missing")
	}
	var issuer models.UserTenant
	if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, issuerID).Take(&issuer).Error != nil || (issuer.Role != "owner" && issuer.Role != "admin") {
		return errors.New("operator_forbidden")
	}
	return nil
}

// PrepareSyntheticInterventionDemo creates only clearly named invented source
// rows and a manual job. Existing rows are validated and left intact.
func PrepareSyntheticInterventionDemo(ctx context.Context, gdb *gorm.DB, tenantID, issuerID string, service *engine.InterventionService) (*SyntheticDemoIDs, error) {
	if gdb == nil || tenantID == "" || issuerID == "" || service == nil || !service.SyntheticConfigured() {
		return nil, errors.New("intervention_unavailable")
	}
	result := &SyntheticDemoIDs{TenantID: tenantID}
	err := gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := interventionOperator(tx, tenantID, issuerID); err != nil {
			return err
		}
		var channels []models.Channel
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND external_id = ?", tenantID, syntheticChannelExternalID).Find(&channels).Error != nil {
			return errors.New("channel_unavailable")
		}
		if len(channels) > 1 {
			return errors.New("fixture_conflict")
		}
		now := time.Now().UTC()
		var channel models.Channel
		if len(channels) == 0 {
			channel = models.Channel{ID: pkg.NewUUID(), TenantID: tenantID, ChannelType: "pancake", Name: "Synthetic intervention demonstration", ExternalID: syntheticChannelExternalID, CredentialsEncrypted: []byte(`{"synthetic":true}`), IsActive: true, IsDemoFixture: true, Metadata: `{}`, CreatedAt: now, UpdatedAt: now}
			if tx.Create(&channel).Error != nil {
				return errors.New("fixture_unavailable")
			}
		} else {
			channel = channels[0]
			if channel.ChannelType != "pancake" || !channel.IsDemoFixture || !channel.IsActive {
				return errors.New("fixture_conflict")
			}
		}
		result.ChannelID = channel.ID
		ensureConversation := func(external, customer string, complaint bool) (string, error) {
			var rows []models.Conversation
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND channel_id = ? AND external_conversation_id = ?", tenantID, channel.ID, external).Find(&rows).Error != nil || len(rows) > 1 {
				return "", errors.New("fixture_conflict")
			}
			var conv models.Conversation
			if len(rows) == 0 {
				conv = models.Conversation{ID: pkg.NewUUID(), TenantID: tenantID, ChannelID: channel.ID, ExternalConversationID: external, CustomerName: customer, Metadata: `{}`, CreatedAt: now, UpdatedAt: now}
				if tx.Create(&conv).Error != nil {
					return "", errors.New("fixture_unavailable")
				}
			} else {
				conv = rows[0]
			}
			if complaint {
				var count int64
				if tx.Model(&models.Message{}).Where("tenant_id = ? AND conversation_id = ?", tenantID, conv.ID).Count(&count).Error != nil {
					return "", errors.New("fixture_unavailable")
				}
				if count == 0 {
					message := models.Message{ID: pkg.NewUUID(), TenantID: tenantID, ConversationID: conv.ID, ExternalMessageID: "ccma-synthetic-complaint-message", SenderType: "customer", Content: "Tôi khiếu nại vì đơn hàng bị chậm và đang chờ phản hồi.", ContentType: "text", Attachments: `[]`, RawData: `{}`, SentAt: now, CreatedAt: now}
					if tx.Create(&message).Error != nil {
						return "", errors.New("fixture_unavailable")
					}
					if tx.Model(&models.Conversation{}).Where("id = ?", conv.ID).Updates(map[string]any{"message_count": 1, "last_message_at": now}).Error != nil {
						return "", errors.New("fixture_unavailable")
					}
				}
			}
			return conv.ID, nil
		}
		var err error
		result.ComplaintConversation, err = ensureConversation("ccma-synthetic-complaint", "Synthetic complaint customer", true)
		if err != nil {
			return err
		}
		result.WaitConversation, err = ensureConversation(syntheticWaitExternalID, "Synthetic waiting customer", false)
		if err != nil {
			return err
		}
		var jobs []models.Job
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND description = ?", tenantID, syntheticJobDescription).Find(&jobs).Error != nil || len(jobs) > 1 {
			return errors.New("fixture_conflict")
		}
		if len(jobs) == 0 {
			input, _ := json.Marshal([]string{channel.ID})
			job := models.Job{ID: pkg.NewUUID(), TenantID: tenantID, Name: "Synthetic intervention review", Description: syntheticJobDescription, JobType: "intervention", InputChannelIDs: string(input), RulesConfig: `{}`, Outputs: `[]`, OutputSchedule: "none", ScheduleType: "manual", IsActive: true, AIProvider: service.Provider, AIModel: service.Model, CreatedAt: now, UpdatedAt: now}
			if tx.Create(&job).Error != nil {
				return errors.New("fixture_unavailable")
			}
			result.JobID = job.ID
		} else {
			job := jobs[0]
			var ids []string
			if json.Unmarshal([]byte(job.InputChannelIDs), &ids) != nil || len(ids) != 1 || ids[0] != channel.ID || job.JobType != "intervention" || job.ScheduleType != "manual" || !job.IsActive || job.AIProvider != service.Provider || job.AIModel != service.Model {
				return errors.New("fixture_conflict")
			}
			result.JobID = job.ID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CompleteSyntheticInterventionDemoData adds one fixed invented customer
// message to the operator-owned empty WAIT_DATA conversation. It is safe to
// retry and never edits or deletes an existing source row.
func CompleteSyntheticInterventionDemoData(ctx context.Context, gdb *gorm.DB, tenantID, issuerID string) (string, error) {
	if gdb == nil || tenantID == "" || issuerID == "" {
		return "", errors.New("intervention_unavailable")
	}
	messageID := ""
	err := gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := interventionOperator(tx, tenantID, issuerID); err != nil {
			return err
		}
		var channel models.Channel
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND external_id = ? AND channel_type = ? AND is_demo_fixture = ?", tenantID, syntheticChannelExternalID, "pancake", true).Take(&channel).Error != nil {
			return errors.New("fixture_missing")
		}
		var conv models.Conversation
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND channel_id = ? AND external_conversation_id = ?", tenantID, channel.ID, syntheticWaitExternalID).Take(&conv).Error != nil || conv.CustomerName != "Synthetic waiting customer" {
			return errors.New("fixture_conflict")
		}
		var existing []models.Message
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND conversation_id = ?", tenantID, conv.ID).Find(&existing).Error != nil {
			return errors.New("fixture_unavailable")
		}
		if len(existing) > 0 {
			if len(existing) != 1 || existing[0].ExternalMessageID != syntheticCompletedMessageID || existing[0].Content != syntheticCompletedMessageText || existing[0].SenderType != "customer" {
				return errors.New("fixture_conflict")
			}
			messageID = existing[0].ID
			return nil
		}
		now := time.Now().UTC()
		message := models.Message{ID: pkg.NewUUID(), TenantID: tenantID, ConversationID: conv.ID, ExternalMessageID: syntheticCompletedMessageID, SenderType: "customer", Content: syntheticCompletedMessageText, ContentType: "text", Attachments: `[]`, RawData: `{}`, SentAt: now, CreatedAt: now}
		if tx.Create(&message).Error != nil {
			return errors.New("fixture_unavailable")
		}
		if tx.Model(&models.Conversation{}).Where("tenant_id = ? AND id = ?", tenantID, conv.ID).Updates(map[string]any{"message_count": 1, "last_message_at": now}).Error != nil {
			return errors.New("fixture_unavailable")
		}
		messageID = message.ID
		return nil
	})
	return messageID, err
}

// SetSyntheticInterventionBudget can only raise an explicit cap. Consumed is
// never updated, even after unknown provider outcomes or process restarts.
func SetSyntheticInterventionBudget(ctx context.Context, gdb *gorm.DB, tenantID, issuerID, jobID string, cap int) (*models.InterventionBudget, error) {
	if gdb == nil || cap < 1 || cap > 1000 {
		return nil, errors.New("budget_cap_invalid")
	}
	var result models.InterventionBudget
	err := gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := interventionOperator(tx, tenantID, issuerID); err != nil {
			return err
		}
		var job models.Job
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, jobID).Take(&job).Error != nil || job.Description != syntheticJobDescription || job.JobType != "intervention" || job.ScheduleType != "manual" || !job.IsActive {
			return errors.New("job_invalid")
		}
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND job_id = ?", tenantID, jobID).Take(&result)
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			result = models.InterventionBudget{ID: pkg.NewUUID(), TenantID: tenantID, JobID: jobID, Cap: cap, Consumed: 0, CreatedAt: time.Now().UTC()}
			if tx.Create(&result).Error != nil {
				return errors.New("budget_unavailable")
			}
			return nil
		}
		if q.Error != nil {
			return errors.New("budget_unavailable")
		}
		if cap < result.Cap || cap < result.Consumed {
			return errors.New("budget_cap_nonmonotone")
		}
		if cap > result.Cap {
			if tx.Model(&models.InterventionBudget{}).Where("id = ? AND cap = ? AND consumed = ?", result.ID, result.Cap, result.Consumed).Update("cap", cap).RowsAffected != 1 {
				return errors.New("budget_conflict")
			}
			result.Cap = cap
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func interventionCLIConnection() (*engine.InterventionService, *gorm.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	service, err := engine.LoadSyntheticInterventionService(cfg, engine.ReadSyntheticInterventionKeyFile)
	if err != nil || service == nil {
		return nil, nil, errors.New("intervention_unavailable")
	}
	if err := db.Connect(cfg.DSN(), cfg.IsProduction()); err != nil {
		return nil, nil, errors.New("database_unavailable")
	}
	if err := db.AutoMigrate(); err != nil {
		db.Close()
		return nil, nil, errors.New("database_migration_unavailable")
	}
	if err := db.EnsureSingleWorkspace(); err != nil {
		db.Close()
		return nil, nil, errors.New("workspace_unavailable")
	}
	return service, db.DB.Session(&gorm.Session{Logger: logger.Discard}), nil
}

// SyntheticInterventionOperator is available only to an operator with server
// CLI access. All outputs are identifiers and bounded state; no key is printed.
func SyntheticInterventionOperator(command string, args []string) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	tenant := fs.String("tenant", "", "tenant UUID")
	issuer := fs.String("issuer", "", "owner/admin user UUID")
	job := fs.String("job", "", "manual synthetic job UUID")
	caseID := fs.String("case", "", "WAIT_DATA case UUID")
	request := fs.String("request", "", "rerun request UUID")
	actor := fs.String("actor", "", "authorized request actor UUID")
	evidence := fs.String("evidence", "", "operator approval reference")
	cap := fs.Int("cap", 0, "monotone total call cap, 1..1000")
	deadline := fs.String("deadline", "", "grant deadline RFC3339, 1 minute to 24 hours ahead")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(*tenant) == "" || strings.TrimSpace(*issuer) == "" {
		return errors.New("tenant_and_issuer_required")
	}
	service, gdb, err := interventionCLIConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var output any
	switch command {
	case "intervention-prepare-demo":
		output, err = PrepareSyntheticInterventionDemo(ctx, gdb, *tenant, *issuer, service)
	case "intervention-set-budget":
		output, err = SetSyntheticInterventionBudget(ctx, gdb, *tenant, *issuer, *job, *cap)
	case "intervention-complete-demo-data":
		var id string
		id, err = CompleteSyntheticInterventionDemoData(ctx, gdb, *tenant, *issuer)
		output = map[string]string{"message_id": id}
	case "intervention-issue-grant":
		var at time.Time
		at, err = time.Parse(time.RFC3339, *deadline)
		if err == nil {
			output, _, err = engine.IssueSyntheticInterventionGrant(ctx, *tenant, *issuer, *actor, *caseID, *request, *evidence, at, service)
		}
	default:
		err = errors.New("unknown_intervention_command")
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(output)
}
