package check

type Runnable interface {
	Run() *Result
}

type WithHint interface {
	Hint() string
}

type WithFix interface {
	Fix()
}

type WithOnError interface {
	OnError()
}

type WithName interface {
	GetName() string
}

type Check interface {
	Runnable
	WithName
}
