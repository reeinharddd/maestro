package db

import "strings"

type upsertCol struct {
	Name string
	Val  any
}

func (d *DB) upsertRow(table, pk string, cols []upsertCol) error {
	var sb strings.Builder

	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (")
	for i, c := range cols {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(c.Name)
	}
	sb.WriteString(") VALUES (")
	for i := range cols {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("?")
	}
	sb.WriteString(") ON CONFLICT(")
	sb.WriteString(pk)
	sb.WriteString(") DO UPDATE SET ")

	needsComma := false
	for _, c := range cols {
		if c.Name == pk {
			continue
		}
		if needsComma {
			sb.WriteString(", ")
		}
		needsComma = true
		sb.WriteString(c.Name)
		sb.WriteString("=excluded.")
		sb.WriteString(c.Name)
	}

	values := make([]any, len(cols))
	for i, c := range cols {
		values[i] = c.Val
	}

	_, err := d.Exec(sb.String(), values...)
	return err
}
