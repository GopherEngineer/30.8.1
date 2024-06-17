package storage

// "Модель" задачи.
type Task struct {
	ID         int
	Opened     int64
	Closed     int64
	AuthorID   int
	AssignedID int
	Title      string
	Content    string
}

// "Модель" пользователя.
type User struct {
	ID   int
	Name string
}

// "Модель" метки.
type Label struct {
	ID   int
	Name string
}

// Interface задаёт контракт на работу с БД.
type Interface interface {
	Tasks() ([]Task, error)
	TaskById(taskId int) (*Task, error)
	TasksByAuthor(authorId int) ([]Task, error)
	TasksByLabel(labelId int) ([]Task, error)
	AddTask(task Task) (int, error)
	AddTasks(tasks []Task) ([]int, error)
	AddTasksBatch(tasks []Task) error
	UpdateTask(task Task) error
	DeleteTask(taskId int) error
}
