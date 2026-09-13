//go:build linux

package companion

import "tws_manager/internal/bt"

func configurePasswordProvider(c *Controller) { bt.ConfigureSudoPasswordProvider(c.Password) }
