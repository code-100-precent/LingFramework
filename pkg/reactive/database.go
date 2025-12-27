package reactive

import (
	"gorm.io/gorm"
)

// ReactiveDB wraps a gorm.DB to provide reactive database operations
type ReactiveDB struct {
	db *gorm.DB
}

// NewReactiveDB creates a new ReactiveDB
func NewReactiveDB(db *gorm.DB) *ReactiveDB {
	return &ReactiveDB{db: db}
}

// Find publishes all records matching the query
func (r *ReactiveDB) Find(dest interface{}, conds ...interface{}) Publisher {
	flow := NewFlow()

	go func() {
		defer flow.Complete()

		result := r.db.Find(dest, conds...)
		if result.Error != nil {
			flow.PublishError(result.Error)
			return
		}

		// Convert result to slice and publish each item
		if slice, ok := dest.([]interface{}); ok {
			for _, item := range slice {
				if err := flow.Publish(item); err != nil {
					return
				}
			}
		} else {
			// Single item
			flow.Publish(dest)
		}
	}()

	return flow
}

// First publishes the first record matching the query
func (r *ReactiveDB) First(dest interface{}, conds ...interface{}) Publisher {
	flow := NewFlow()

	go func() {
		defer flow.Complete()

		result := r.db.First(dest, conds...)
		if result.Error != nil {
			flow.PublishError(result.Error)
			return
		}

		flow.Publish(dest)
	}()

	return flow
}

// Where adds a where clause to the query
func (r *ReactiveDB) Where(query interface{}, args ...interface{}) *ReactiveDB {
	return &ReactiveDB{db: r.db.Where(query, args...)}
}

// Order adds an order clause to the query
func (r *ReactiveDB) Order(value interface{}) *ReactiveDB {
	return &ReactiveDB{db: r.db.Order(value)}
}

// Limit adds a limit clause to the query
func (r *ReactiveDB) Limit(limit int) *ReactiveDB {
	return &ReactiveDB{db: r.db.Limit(limit)}
}

// Offset adds an offset clause to the query
func (r *ReactiveDB) Offset(offset int) *ReactiveDB {
	return &ReactiveDB{db: r.db.Offset(offset)}
}

// Create publishes the created record
func (r *ReactiveDB) Create(value interface{}) Publisher {
	flow := NewFlow()

	go func() {
		defer flow.Complete()

		result := r.db.Create(value)
		if result.Error != nil {
			flow.PublishError(result.Error)
			return
		}

		flow.Publish(value)
	}()

	return flow
}

// Update publishes the updated record
func (r *ReactiveDB) Update(column string, value interface{}) Publisher {
	flow := NewFlow()

	go func() {
		defer flow.Complete()

		result := r.db.Update(column, value)
		if result.Error != nil {
			flow.PublishError(result.Error)
			return
		}

		flow.Publish(map[string]interface{}{
			"rows_affected": result.RowsAffected,
		})
	}()

	return flow
}

// Delete publishes the deletion result
func (r *ReactiveDB) Delete(value interface{}, conds ...interface{}) Publisher {
	flow := NewFlow()

	go func() {
		defer flow.Complete()

		result := r.db.Delete(value, conds...)
		if result.Error != nil {
			flow.PublishError(result.Error)
			return
		}

		flow.Publish(map[string]interface{}{
			"rows_affected": result.RowsAffected,
		})
	}()

	return flow
}

// Raw executes a raw SQL query and publishes the results
func (r *ReactiveDB) Raw(sql string, values ...interface{}) Publisher {
	flow := NewFlow()

	go func() {
		defer flow.Complete()

		rows, err := r.db.Raw(sql, values...).Rows()
		if err != nil {
			flow.PublishError(err)
			return
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				flow.PublishError(err)
				return
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				row[col] = values[i]
			}

			if err := flow.Publish(row); err != nil {
				return
			}
		}
	}()

	return flow
}
