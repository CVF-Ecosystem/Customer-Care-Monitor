# R096 actual transport failure and consolidated source repair review

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/m2_worker
Reviewer: Codex /root

Both actual application attempts consumed1parentphysicalslot each andDBbudget1then2; actualproviderHTTPstatus0/noresponse,TRANSPORT_UNKNOWN. CaseUSAGE_UNKNOWN/PROVIDER_UNKNOWN,attemptUNKNOWN,usagetokensNULL,proposals0. AppHTTP200 is only honest savedfailurecase, never AI/proposal success. Ownerphysical2/2 permanentlyspent, no refund/retry/third effect. Root's reviewercall followsworker withindefinedcampaign andsamehashes. No realproviderresponse/livegovernance/outputvalidationproof established.

F1 reproducedsourceharnessdefect: Go1.27 installedprimarysource Transport.Clone initializesHTTP2 andcopiesTLS.NextProtos[h2,http/1.1]. ForceAttemptHTTP2false/emptyTLSNextProto doesnotclearadvertisedALPN. RootindependentloopbackTLS dualprotocol fixture reproducesmalformedHTTPresponsebefore andHTTP1.1/200afterexplicitNextProtos[http/1.1]; rawproof docs/reviews/probes/r096_root/tls_repro_stdout.json andstderr. No provider/DB/AI mockeddecision inthisprotocoltest. Actualremoteerrorwasnottype-captured, so matchingremote cause is inference/unproved. RootownsHTTP1directive/reviewmiss, not evidenceAlibaba failed.

Same-seedrepair1: SolcorrectisolatedTLS.NextProtosonly,preservecertificateverification andallcaps/model/prompt/source/parentmarkers, addpureloopbackTLSprotocol regression. One remainingworkerGo4 selectedtest with-work mayretainexactcandidatebinary; Root3/3Goexhausted after independentrepro, source/static/Python review only. No newprovider/key/modeltuning/fixture orliveeffect. Reviewrequiredafterrepair; boundedSourcefix maypass butR096live/FREEZE remainsblocked. Preparecorrectedsource-boundfreshbudgetapprovalpacket onlyafterreview/cleanup/commits. No additionalcallauthority untilownergrantsnewbudget asAGENTS boundaryescalation requires.
