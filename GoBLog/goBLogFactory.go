//GoBLogFactory
package GoBLog

import (
	"fmt"
	_ "log"
)

type GoBLogFactory struct {
}

func init() {
	fmt.Println("init JLogFactory!")

}

func NewGOBLog() (newlog *GOBLog) {

	return nil
}
