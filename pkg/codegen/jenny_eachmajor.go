package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/grafana/codejen"
	"github.com/grafana/grafana/pkg/kindsys"
)

// LatestMajorsOrXJenny returns a jenny that repeats the input for the latest in each major version,
func LatestMajorsOrXJenny(parentdir string, inner codejen.OneToOne[SchemaForGen]) OneToMany {
	if inner == nil {
		panic("inner jenny must not be nil")
	}

	return &lmox{
		parentdir: parentdir,
		inner:     inner,
	}
}

type lmox struct {
	parentdir string
	inner     codejen.OneToOne[SchemaForGen]
}

func (j *lmox) JennyName() string {
	return "LatestMajorsOrXJenny"
}

func (j *lmox) Generate(decl *DeclForGen) (codejen.Files, error) {
	if decl.IsRaw() {
		return nil, nil
	}
	comm := decl.Properties.Common()
	sfg := SchemaForGen{
		Name:    comm.Name,
		IsGroup: comm.LineageIsGroup,
	}

	do := func(sfg SchemaForGen, infix string) (codejen.Files, error) {
		f, err := j.inner.Generate(sfg)
		if err != nil {
			return nil, fmt.Errorf("%s jenny failed on %s schema for %s: %w", j.inner.JennyName(), sfg.Schema.Version(), decl.Properties.Common().Name, err)
		}
		if f == nil || !f.Exists() {
			return nil, nil
		}

		f.RelativePath = filepath.Join(j.parentdir, comm.MachineName, infix, f.RelativePath)
		f.From = append(f.From, j)
		return codejen.Files{*f}, nil
	}

	if comm.Maturity.Less(kindsys.MaturityStable) {
		sfg.Schema = decl.Lineage().Latest()
		return do(sfg, "x")
	}

	var fl codejen.Files
	for sch := decl.Lineage().First(); sch != nil; sch.Successor() {
		sfg.Schema = sch.LatestInMajor()
		files, err := do(sfg, fmt.Sprintf("v%v", sch.Version()[0]))
		if err != nil {
			return nil, err
		}
		fl = append(fl, files...)
	}
	if fl.Validate() != nil {
		return nil, fl.Validate()
	}
	return fl, nil
}
