package types

var (
	CertX      = "cert"
	ExecerCert = []byte(CertX)
	actionName = map[string]int32{
		"New":    CertActionNew,
		"Update": CertActionUpdate,
		"Normal": CertActionNormal,
	}
)
