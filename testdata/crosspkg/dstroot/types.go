package dstroot

import "github.com/seitarof/gen-dto/testdata/crosspkg/dstnested"

type Notification struct {
	Recipient *dstnested.Recipient
}
