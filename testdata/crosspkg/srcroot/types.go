package srcroot

import "github.com/seitarof/gen-dto/testdata/crosspkg/srcnested"

type Notification struct {
	Recipient *srcnested.Recipient
}
