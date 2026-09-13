//go:build linux

package companion

import "nothing_helper/internal/bt"

func configurePasswordProvider(c *Controller) { bt.ConfigureSudoPasswordProvider(c.Password) }
