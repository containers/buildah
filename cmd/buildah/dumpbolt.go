package main

import (
	"fmt"
	"strings"

	bolt "github.com/etcd-io/bbolt"
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
			if value[i] <= 32 || value[i] >= 127 {
				b.WriteString(fmt.Sprintf("\\%03o", value[i]))
			} else {
				b.WriteByte(value[i])
			}
		}
		return b.String()
	}

	return db.View(func(tx *bolt.Tx) error {
		var dumpBucket func(string, []byte, *bolt.Bucket) error
		dumpBucket = func(indent string, name []byte, b *bolt.Bucket) error {
			var subs [][]byte
			indentMore := "  "
			fmt.Printf("%s%s:\n", indent, encode(name))
			err := b.ForEach(func(k, v []byte) (err error) {
				if v == nil {
					subs = append(subs, k)
				} else {
					_, err = fmt.Printf("%s%s: %s\n", indent+indentMore, encode(k), encode(v))
				}
				return err
			})
			if err != nil {
				return err
			}
			for _, sub := range subs {
				subbucket := b.Bucket(sub)
				if err = dumpBucket(indent+indentMore, sub, subbucket); err != nil {
					return err
				}
			}
			return err
		}
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error { return dumpBucket("", name, b) })
	})
}

func init() {
	rootCmd.AddCommand(dumpBoltCommand)
}
