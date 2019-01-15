// GoBLogFactory
package GoBLog

import (
	"fmt"
)

func init() {
	fmt.Println("init LogFactory!")

}

type LogFactory struct {
}

func (this *LogFactory) NewLogger() (newlogger *GoBLogger) {

	return nil
}
