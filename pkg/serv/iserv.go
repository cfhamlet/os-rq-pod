package serv

// IServ TODO
type IServ interface {
	OnStart() error
	OnStop() error
}
