package subcmd

import (
	"github.com/Lupino/go-periodic"
	"log"
)

// SubmitJob cli submit
func SubmitJob(entryPoint, xor, funcName, name string, opts map[string]string) {
	c := periodic.NewClient()
	if err := c.Connect(entryPoint, xor); err != nil {
		log.Fatal(err)
	}
	if err := c.SubmitJob(funcName, name, opts); err != nil {
		log.Fatal(err)
	}
	log.Printf("Submit Job[%s] success.\n", name)
}
