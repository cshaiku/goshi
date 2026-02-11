package app

type Capability string

const (
	CapFSRead  Capability = "FS_READ"
	CapFSWrite Capability = "FS_WRITE"
)

type Capabilities struct {
	granted map[Capability]bool
}

func NewCapabilities() *Capabilities {
	return &Capabilities{
		granted: make(map[Capability]bool),
	}
}

func (c *Capabilities) Grant(cap Capability) {
	c.granted[cap] = true
}

func (c *Capabilities) Has(cap Capability) bool {
	return c.granted[cap]
}
