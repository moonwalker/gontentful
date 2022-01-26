package gontentful

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type PGMatViews struct {
	Schema *PGSQLSchema
}

type PGMatView struct {
	Locales  []*Locale
	Function *PGSQLProcedure
}

const xthreads = 10

func NewPGMatViews(schema *PGSQLSchema) *PGMatViews {
	return &PGMatViews{
		Schema: schema,
	}
}

func (s *PGMatViews) Exec(databaseURL string) error {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}
	tmpl, err := template.New("").Funcs(funcMap).Parse(pgRefreshMatViewsTemplate)

	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s.Schema)
	if err != nil {
		return err
	}

	db, err := sqlx.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if s.Schema.SchemaName != "" {
		// set schema in use
		_, err = db.Exec(fmt.Sprintf("SET search_path='%s'", s.Schema.SchemaName))
		if err != nil {
			return err
		}
	}

	// ioutil.WriteFile("/tmp/matViews", buff.Bytes(), 0644)

	_, err = db.Exec(buff.String())
	if err != nil {
		return err
	}

	return nil
}

func (s *PGMatViews) ExecOneByOne(databaseURL string, schemaName string) error {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}
	tmpl, err := template.New("").Funcs(funcMap).Parse(pgRefreshMatViewsTemplate1)
	if err != nil {
		return err
	}

	params := make([]*PGMatView, 0)
	for _, f := range s.Schema.Functions {
		params = append(params, &PGMatView{
			Locales:  s.Schema.Locales,
			Function: f,
		})
	}

	var ch = make(chan *PGMatView, len(params)) // This number 50 can be anything as long as it's larger than xthreads
	var wg sync.WaitGroup

	// This starts xthreads number of goroutines that wait for something to do
	wg.Add(xthreads)
	for i := 0; i < xthreads; i++ {
		go func() {
			for {
				a, ok := <-ch
				if !ok { // if there is nothing to do and the channel has been closed then end the goroutine
					wg.Done()
					return
				}
				createMatView(tmpl, a, databaseURL, schemaName) // do the thing
			}
		}()
	}

	// Now the jobs can be added to the channel, which is used as a queue
	for _, mv := range params {
		ch <- mv // add mv to the queue
	}

	close(ch) // This tells the goroutines there's nothing else to do
	wg.Wait() // Wait for the threads to finish

	return nil
}

func createMatView(tmpl *template.Template, mv *PGMatView, databaseURL string, schemaName string) error {
	db, err := sqlx.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if schemaName != "" {
		// set schema in use
		_, err = db.Exec(fmt.Sprintf("SET search_path='%s'", schemaName))
		if err != nil {
			return err
		}
	}

	var buff bytes.Buffer

	err = tmpl.Execute(&buff, mv)
	if err != nil {
		return err
	}

	_, err = db.Exec(buff.String())
	if err != nil {
		return err
	}

	return nil
}
