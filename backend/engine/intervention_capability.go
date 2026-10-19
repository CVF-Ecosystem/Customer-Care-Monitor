package engine

// SyntheticConfigured reports process configuration only. It does not assert
// tenant permission, fixture eligibility, available budget, or provider health.
func (s *InterventionService) SyntheticConfigured() bool {
	return s.configured() && s.PolicyVersion == "ccma.synthetic.intervention.v1" && s.Provider != "" && s.Model != "" && s.ConfigRef != ""
}
