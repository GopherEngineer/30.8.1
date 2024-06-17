package postgres

import (
	"context"
	"skillfactory/30.8.1/pkg/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Хранилище данных.
type Storage struct {
	pool *pgxpool.Pool
}

// Конструктор, принимает строку подключения к БД.
func New(constr string) (*Storage, error) {
	pool, err := pgxpool.New(context.Background(), constr)
	if err != nil {
		return nil, err
	}
	s := Storage{
		pool: pool,
	}
	return &s, nil
}

// Tasks возвращает список задач из БД.
func (s *Storage) Tasks() ([]storage.Task, error) {
	rows, err := s.pool.Query(context.Background(), `
		SELECT 
			id,
			opened,
			closed,
			author_id,
			assigned_id,
			title,
			content
		FROM tasks
		ORDER BY id;
	`)
	if err != nil {
		return nil, err
	}
	var tasks []storage.Task

	// итерирование по результату выполнения запроса
	// и сканирование каждой строки в переменную
	for rows.Next() {
		var t storage.Task
		err = rows.Scan(
			&t.ID,
			&t.Opened,
			&t.Closed,
			&t.AuthorID,
			&t.AssignedID,
			&t.Title,
			&t.Content,
		)
		if err != nil {
			return nil, err
		}

		// добавление переменной в массив результатов
		tasks = append(tasks, t)
	}

	// ВАЖНО не забыть проверить rows.Err()
	return tasks, rows.Err()
}

// TaskById возвращает задачу по её ID.
func (s *Storage) TaskById(taskId int) (*storage.Task, error) {
	var t storage.Task

	err := s.pool.QueryRow(context.Background(), `
		SELECT 
			id,
			opened,
			closed,
			author_id,
			assigned_id,
			title,
			content
		FROM tasks
		WHERE id = $1;
	`,
		taskId,
	).Scan(
		&t.ID,
		&t.Opened,
		&t.Closed,
		&t.AuthorID,
		&t.AssignedID,
		&t.Title,
		&t.Content,
	)
	if err != nil {
		return nil, err
	}

	return &t, err
}

// TasksByAuthor возвращает слайс задач по ID автора.
func (s *Storage) TasksByAuthor(authorId int) ([]storage.Task, error) {
	rows, err := s.pool.Query(context.Background(), `
		SELECT 
			id,
			opened,
			closed,
			author_id,
			assigned_id,
			title,
			content
		FROM tasks
		WHERE author_id = $1
		ORDER BY id;
	`,
		authorId,
	)
	if err != nil {
		return nil, err
	}
	var tasks []storage.Task

	for rows.Next() {
		var t storage.Task
		err = rows.Scan(
			&t.ID,
			&t.Opened,
			&t.Closed,
			&t.AuthorID,
			&t.AssignedID,
			&t.Title,
			&t.Content,
		)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

// TasksByLabel возвращает слайс задач по ID метки.
func (s *Storage) TasksByLabel(labelId int) ([]storage.Task, error) {
	rows, err := s.pool.Query(context.Background(), `
		SELECT * FROM tasks
		WHERE id IN (
			SELECT task_id FROM tasks_labels
			WHERE label_id = $1
		)
		ORDER BY id;
	`,
		labelId,
	)
	if err != nil {
		return nil, err
	}
	var tasks []storage.Task

	for rows.Next() {
		var t storage.Task
		err = rows.Scan(
			&t.ID,
			&t.Opened,
			&t.Closed,
			&t.AuthorID,
			&t.AssignedID,
			&t.Title,
			&t.Content,
		)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

// AddTask создаёт новую задачу и возвращает её id.
func (s *Storage) AddTask(t storage.Task) (int, error) {
	var id int
	err := s.pool.QueryRow(context.Background(), `
		INSERT INTO tasks (title, content)
		VALUES ($1, $2) RETURNING id;
	`,
		t.Title,
		t.Content,
	).Scan(&id)
	return id, err
}

// AddTasks создаёт новые задачи и возвращает слайс ID созданых задач.
// Пример работы с транзакцией.
func (s *Storage) AddTasks(tasks []storage.Task) ([]int, error) {
	var ids []int

	// Простой базовый контект без таймаута.
	ctx := context.Background()

	// Начинаем транзакцию с базой данных.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Проходим по слайсу задач и отправляем задачу на создание в базу данных.
	for _, task := range tasks {
		var id int
		err := tx.QueryRow(ctx, `
			INSERT INTO tasks (title, content)
			VALUES ($1, $2) RETURNING id;
		`,
			task.Title,
			task.Content,
		).Scan(&id)
		if err != nil {
			// в случае неудачного выполнения запроса откатываем изменения
			// и возвращаем полученную ошибку
			tx.Rollback(ctx)
			return nil, err
		}
		ids = append(ids, id)
	}

	// Применяем все изменения в базе данных.
	err = tx.Commit(ctx)

	// Возвращаем слайс ID созданных задач.
	return ids, err
}

// AddTasksBatch создаёт новые задачи и в случае неудачи возвращает ошибку.
// Пример работы с партией запросов.
func (s *Storage) AddTasksBatch(tasks []storage.Task) error {
	batch := pgx.Batch{}

	for _, task := range tasks {
		batch.Queue(`
			INSERT INTO tasks (title, content)
			VALUES ($1, $2);
		`,
			task.Title,
			task.Content,
		)
	}

	results := s.pool.SendBatch(context.Background(), &batch)
	defer results.Close()

	_, err := results.Query()

	return err
}

// UpdateTask обновляет задачу принимая в качестве агрумента экземпляр структуры Task.
func (s *Storage) UpdateTask(task storage.Task) error {
	_, err := s.pool.Query(context.Background(), `
		UPDATE tasks
		SET (opened, closed, author_id, assigned_id, title, content) = ($2, $3, $4, $5, $6, $7)
		WHERE id = $1;
	`,
		task.ID,
		task.Opened,
		task.Closed,
		task.AuthorID,
		task.AssignedID,
		task.Title,
		task.Content,
	)
	return err
}

// DeleteTask удаляет задачу по ID.
func (s *Storage) DeleteTask(taskId int) error {
	_, err := s.pool.Query(context.Background(), `
		DELETE FROM tasks
		WHERE id = $1;
	`,
		taskId,
	)
	return err
}
