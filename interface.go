package periodic

type ClientI interface {
	Ping() bool
	SubmitJob(string, string, map[string]interface{}) error
	RunJob(string, string, map[string]interface{}) (error, []byte)
	Status() ([][]string, error)
	DropFunc(string) error
	RemoveJob(string, string) error
	Close()
}
