package subcmd

import (
	"github.com/Lupino/go-periodic"
	"github.com/Lupino/go-periodic/protocol"
	"log"
)

// DropFunc cli drop
func DropFunc(entryPoint string, param protocol.RSAConnParam, funcName string, auth ...protocol.ClientAuth) {
	c := periodic.NewClient()
	if len(auth) > 0 {
		if err := c.SetAuth(auth[0].Name, auth[0].Token); err != nil {
			log.Fatal(err)
		}
	}
	if err := c.Connect(entryPoint, param); err != nil {
		log.Fatal(err)
	}
	if err := c.DropFunc(funcName); err != nil {
		log.Fatal(err)
	}
	log.Printf("Drop Func[%s] success.\n", funcName)
}
