package subcmd

import (
	"github.com/Lupino/go-periodic"
	"github.com/Lupino/go-periodic/protocol"
	"log"
)

// SubmitJob cli submit
func SubmitJob(entryPoint string, param protocol.RSAConnParam, funcName, name string, opts map[string]interface{}, auth ...protocol.ClientAuth) {
	c := periodic.NewClient()
	if len(auth) > 0 {
		if err := c.SetAuth(auth[0].Name, auth[0].Token); err != nil {
			log.Fatal(err)
		}
	}
	if err := c.Connect(entryPoint, param); err != nil {
		log.Fatal(err)
	}
	if err := c.SubmitJob(funcName, name, opts); err != nil {
		log.Fatal(err)
	}
	log.Printf("Submit Job[%s] success.\n", name)
}
