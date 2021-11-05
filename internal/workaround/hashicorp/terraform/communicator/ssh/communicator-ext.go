package ssh

import (
	"net"

	"github.com/hashicorp/terraform/terraform"
)

func NewNoPty(s *terraform.InstanceState) (*Communicator, error) {
	c, err := New(s)
	if c != nil && c.config != nil {
		c.config.noPty = true
	}
	return c, err
}

func (c *Communicator) Dial(n string, addr string) (net.Conn, error) {
	return c.client.Dial(n, addr)
}
