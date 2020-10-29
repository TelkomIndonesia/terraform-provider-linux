package ssh

import "github.com/hashicorp/terraform/terraform"

func NewNoPty(s *terraform.InstanceState) (*Communicator, error) {
	c, err := New(s)
	if c != nil && c.config != nil {
		c.config.noPty = true
	}
	return c, err
}
