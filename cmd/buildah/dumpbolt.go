package main

import (
	"fmt"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	dumpBoltDescription = `Dumps a bolt database. The output format should not be depended upon.`
	dumpBoltCommand     = &cobra.Command{
		Use:     "dumpbolt",
		Short:   "Dump a bolt database",
		Long:    dumpBoltDescription,
		RunE:    dumpBoltCmd,
		Example: "DATABASE",
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
	}
)

func dumpBoltCmd(c *cobra.Command, args []string) error {
	db, err := bolt.Open(args[0], 0600, &bolt.Options{ReadOnly: true})
	if err != nil {
		return errors.Wrapf(err, "error opening database %q", args[0])
	}
	defer db.Close()

	encode := func(value []byte) string {
		var b strings.Builder
		for i := range value {
			if value[i] <= 32 || value[i] >= 127 || value[i] == 34 || value[i] == 61 {
				b.WriteString(fmt.Sprintf("\\%03o", value[i]))
			} else {
				b.WriteByte(value[i])
			}
		}
		return b.String()
	}

	return db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			fmt.Printf("[%q]\n", encode(name))
			return b.ForEach(func(k, v []byte) (err error) {
				_, err = fmt.Printf(" %q = %q\n", encode(k), encode(v))
				return err
			})
		})
	})
}

func init() {
	rootCmd.AddCommand(dumpBoltCommand)
}
