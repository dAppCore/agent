package core

import newcore "dappco.re/go"

type Action = newcore.Action
type ActionHandler = newcore.ActionHandler
type App = newcore.App
type AtomicPointer[T any] = newcore.AtomicPointer[T]
type Cli = newcore.Cli
type Command = newcore.Command
type CommandAction = newcore.CommandAction
type Core = newcore.Core
type CoreOption = newcore.CoreOption
type Embed = newcore.Embed
type Fs = newcore.Fs
type Lock = newcore.Lock
type Log = newcore.Log
type Message = newcore.Message
type Mutex = newcore.Mutex
type Once = newcore.Once
type Option = newcore.Option
type Options = newcore.Options
type Process = newcore.Process
type Query = newcore.Query
type Registry[T any] = newcore.Registry[T]
type Result = newcore.Result
type RWMutex = newcore.RWMutex
type Service = newcore.Service
type ServiceRuntime[T any] = newcore.ServiceRuntime[T]
type Startable = newcore.Startable
type Stoppable = newcore.Stoppable
type Translator = newcore.Translator

var As = newcore.As
var Atoi = newcore.Atoi
var CleanPath = newcore.CleanPath
var Concat = newcore.Concat
var Contains = newcore.Contains
var E = newcore.E
var Env = newcore.Env
var ErrorJoin = newcore.ErrorJoin
var Exit = newcore.Exit
var FormatInt = newcore.FormatInt
var HasPrefix = newcore.HasPrefix
var HasSuffix = newcore.HasSuffix
var ID = newcore.ID
var Itoa = newcore.Itoa
var Is = newcore.Is
var IsDigit = newcore.IsDigit
var IsLetter = newcore.IsLetter
var IsSpace = newcore.IsSpace
var JSONMarshal = newcore.JSONMarshal
var JSONMarshalString = newcore.JSONMarshalString
var JSONUnmarshal = newcore.JSONUnmarshal
var JSONUnmarshalString = newcore.JSONUnmarshalString
var Join = newcore.Join
var JoinPath = newcore.JoinPath
var Lower = newcore.Lower
var New = newcore.New
var NewBuffer = newcore.NewBuffer
var NewBuilder = newcore.NewBuilder
var NewError = newcore.NewError
var NewOptions = newcore.NewOptions
var NewReader = newcore.NewReader
var Operation = newcore.Operation
var Path = newcore.Path
var PathBase = newcore.PathBase
var PathDir = newcore.PathDir
var PathExt = newcore.PathExt
var PathGlob = newcore.PathGlob
var PathIsAbs = newcore.PathIsAbs
var PathJoin = newcore.PathJoin
var Print = newcore.Print
var Println = newcore.Println
var ReadAll = newcore.ReadAll
var Replace = newcore.Replace
var SHA256 = newcore.SHA256
var Security = newcore.Security
var Split = newcore.Split
var SplitN = newcore.SplitN
var Sprint = newcore.Sprint
var Sprintf = newcore.Sprintf
var ParseInt = newcore.ParseInt
var Trim = newcore.Trim
var TrimPrefix = newcore.TrimPrefix
var TrimSuffix = newcore.TrimSuffix
var Upper = newcore.Upper
var ValidateName = newcore.ValidateName
var URLParse = newcore.URLParse
var URLEncode = newcore.URLEncode
var Warn = newcore.Warn
var WithName = newcore.WithName
var WithOption = newcore.WithOption
var WithService = newcore.WithService
var Wrap = newcore.Wrap

func NewRegistry[T any]() *Registry[T] {
	return newcore.NewRegistry[T]()
}

func NewServiceRuntime[T any](c *Core, opts T) *ServiceRuntime[T] {
	return newcore.NewServiceRuntime(c, opts)
}
