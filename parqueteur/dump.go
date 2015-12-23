package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kostya-sh/parquet-go/parquet"
	"github.com/kostya-sh/parquet-go/parquetformat"
)

var cmdDump = &Command{
	Name: "dump",
	Help: "dump content of a parquet file",
}

// TODO: support various formats, e.g. CSV, fixed width, pretty print, JSON, etc
var dumpColumn string
var showLevels bool

func init() {
	cmdDump.Run = runDump

	// TODO: better usage message
	cmdDump.Flag.StringVar(&dumpColumn, "c", "", "dump content of the named `column`")
	cmdDump.Flag.BoolVar(&showLevels, "levels", false, "dump repetition and definition levels along with the column values")
}

func runDump(cmd *Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("No files")
	}

	r, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer r.Close()

	m, err := parquet.ReadFileMetaData(r)
	if err != nil {
		return err
	}
	schema, err := parquet.SchemaFromFileMetaData(m)
	if err != nil {
		return err
	}
	cs := schema.ColumnByName(dumpColumn)
	if cs == nil {
		return fmt.Errorf("no column named '%s' in schema", dumpColumn)
	}

	for i, rg := range m.RowGroups {
		var cc *parquetformat.ColumnChunk
		for _, c := range rg.Columns {
			if strings.Join(c.MetaData.PathInSchema, ".") == dumpColumn {
				cc = c
			}
		}
		if cc == nil {
			return fmt.Errorf("no column named '%s' in rowgroup %d", dumpColumn, i)
		}
		cr, err := parquet.NewBooleanColumnChunkReader(r, cs, cc)
		if err != nil {
			return err
		}
		for cr.Next() {
			levels := cr.Levels()
			value := cr.Boolean()
			notNull := levels.D == cs.MaxLevels.D
			if notNull {
				fmt.Print(value)
			}
			// TODO: consider customizing null value via command lines
			if showLevels {
				if notNull {
					fmt.Printf(" ")
				}
				fmt.Printf("(D:%d; R:%d)", levels.D, levels.R)
			}
			fmt.Println()

		}
		if cr.Err() != nil {
			return cr.Err()
		}
	}

	return nil
}
