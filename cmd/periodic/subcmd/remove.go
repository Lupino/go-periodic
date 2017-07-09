package subcmd

import (
	"github.com/Lupino/go-periodic"
	"log"
)

// RemoveJob cli remove
func RemoveJob(entryPoint, xor, funcName, name string) {
	c := periodic.NewClient()
	if err := c.Connect(entryPoint, xor); err != nil {
		log.Fatal(err)
	}
	if err := c.RemoveJob(funcName, name); err != nil {
		log.Fatal(err)
	}
	log.Printf("Remove Job[%s] success.\n", name)
}
