package golang

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/metadata"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

type QueryValue struct {
	Emit        bool
	EmitPointer bool
	Name        string
	DBName      string // The name of the field in the database. Only set if Struct==nil.
	Struct      *Struct
	Typ         string
	SQLDriver   opts.SQLDriver

	// Column is kept so late in the generation process around to differentiate
	// between mysql slices and pg arrays
	Column *plugin.Column
}

func (v QueryValue) EmitStruct() bool {
	return v.Emit
}

func (v QueryValue) IsStruct() bool {
	return v.Struct != nil
}

func (v QueryValue) IsPointer() bool {
	return v.EmitPointer && v.Struct != nil
}

func (v QueryValue) isEmpty() bool {
	return v.Typ == "" && v.Name == "" && v.Struct == nil
}

type Argument struct {
	Name string
	Type string
}

func (v QueryValue) Pair() string {
	var out []string
	for _, arg := range v.Pairs() {
		out = append(out, arg.Name+" "+arg.Type)
	}
	return strings.Join(out, ",")
}

// Return the argument name and type for query methods. Should only be used in
// the context of method arguments.
func (v QueryValue) Pairs() []Argument {
	if v.isEmpty() {
		return nil
	}
	if !v.EmitStruct() && v.IsStruct() {
		var out []Argument
		for _, f := range v.Struct.Fields {
			out = append(out, Argument{
				Name: escape(toLowerCase(f.Name)),
				Type: f.Type,
			})
		}
		return out
	}
	return []Argument{
		{
			Name: escape(v.Name),
			Type: v.DefineType(),
		},
	}
}

func (v QueryValue) SlicePair() string {
	if v.isEmpty() {
		return ""
	}
	return v.Name + " []" + v.DefineType()
}

func (v QueryValue) Type() string {
	if v.Typ != "" {
		return v.Typ
	}
	if v.Struct != nil {
		return v.Struct.Name
	}
	panic("no type for QueryValue: " + v.Name)
}

func (v *QueryValue) DefineType() string {
	t := v.Type()
	if v.IsPointer() {
		return "*" + t
	}
	return t
}

func (v *QueryValue) ReturnName() string {
	if v.IsPointer() {
		return "&" + escape(v.Name)
	}
	return escape(v.Name)
}

func (v QueryValue) UniqueFields() []Field {
	seen := map[string]struct{}{}
	fields := make([]Field, 0, len(v.Struct.Fields))

	for _, field := range v.Struct.Fields {
		if _, found := seen[field.Name]; found {
			continue
		}
		seen[field.Name] = struct{}{}
		fields = append(fields, field)
	}

	return fields
}

func (v QueryValue) Params() string {
	if v.isEmpty() {
		return ""
	}
	var out []string
	if v.Struct == nil {
		if !v.Column.IsSqlcSlice && strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array("+escape(v.Name)+")")
		} else {
			out = append(out, escape(v.Name))
		}
	} else {
		for _, f := range v.Struct.Fields {
			if !f.HasSqlcSlice() && strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array("+escape(v.VariableForField(f))+")")
			} else {
				out = append(out, escape(v.VariableForField(f)))
			}
		}
	}
	if len(out) <= 3 {
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

func (v QueryValue) ColumnNames() []string {
	if v.Struct == nil {
		return []string{v.DBName}
	}
	names := make([]string, len(v.Struct.Fields))
	for i, f := range v.Struct.Fields {
		names[i] = f.DBName
	}
	return names
}

func (v QueryValue) ColumnNamesAsGoSlice() string {
	if v.Struct == nil {
		return fmt.Sprintf("[]string{%q}", v.DBName)
	}
	escapedNames := make([]string, len(v.Struct.Fields))
	for i, f := range v.Struct.Fields {
		if f.Column != nil && f.Column.OriginalName != "" {
			escapedNames[i] = fmt.Sprintf("%q", f.Column.OriginalName)
		} else {
			escapedNames[i] = fmt.Sprintf("%q", f.DBName)
		}
	}
	return "[]string{" + strings.Join(escapedNames, ", ") + "}"
}

// When true, we have to build the arguments to q.db.QueryContext in addition to
// munging the SQL
func (v QueryValue) HasSqlcSlices() bool {
	if v.Struct == nil {
		return v.Column != nil && v.Column.IsSqlcSlice
	}
	for _, v := range v.Struct.Fields {
		if v.Column.IsSqlcSlice {
			return true
		}
	}
	return false
}

func (v QueryValue) HasNullableEmbed(emitEmbedNullableLeftJoin bool) bool {
	if !emitEmbedNullableLeftJoin || v.Struct == nil {
		return false
	}
	for _, f := range v.Struct.Fields {
		if len(f.EmbedFields) > 0 && f.EmbedIsNullable {
			return true
		}
	}
	return false
}

func (v QueryValue) Scan() string {
	return v.ScanWithOptions(false)
}

// Generate temporary variable declarations for nullable embeds
func (v QueryValue) NullableEmbedTempVars() []string {
	var vars []string
	if v.Struct == nil {
		return vars
	}
	for _, f := range v.Struct.Fields {
		if len(f.EmbedFields) > 0 && f.EmbedIsNullable {
			for _, embed := range f.EmbedFields {
				// All embedded fields need to be nullable types when the embed is nullable
				nullableType := makeNullableType(embed.Type)
				varName := fmt.Sprintf("var %sEmbed%s%s %s", v.Name, f.Name, embed.Name, nullableType)
				vars = append(vars, varName)
			}
		}
	}
	return vars
}

// Convert a type to its nullable equivalent
func makeNullableType(typ string) string {
	switch typ {
	case "string":
		return "sql.NullString"
	case "int", "int32":
		return "sql.NullInt32"
	case "int64":
		return "sql.NullInt64"
	case "float32":
		return "sql.NullFloat64"
	case "float64":
		return "sql.NullFloat64"
	case "bool":
		return "sql.NullBool"
	case "time.Time":
		return "sql.NullTime"
	case "[]byte":
		return "[]byte" // []byte can already handle NULL
	default:
		// For custom types, keep as-is but they should already be nullable
		if strings.HasPrefix(typ, "sql.Null") {
			return typ
		}
		// This might need adjustment based on the actual types used
		return typ
	}
}

// Extract value from a nullable type variable
func extractNullableValue(varName string, originalType string) string {
	switch originalType {
	case "string":
		return varName + ".String"
	case "int", "int32":
		return fmt.Sprintf("int32(%s.Int32)", varName)
	case "int64":
		return varName + ".Int64"
	case "float32":
		return fmt.Sprintf("float32(%s.Float64)", varName)
	case "float64":
		return varName + ".Float64"
	case "bool":
		return varName + ".Bool"
	case "time.Time":
		return varName + ".Time"
	case "[]byte":
		return varName // []byte doesn't need extraction
	default:
		// For sql.Null types, assume they have a field that matches the type name
		if strings.HasPrefix(originalType, "sql.Null") {
			return varName // Already a nullable type, use as-is
		}
		// For other types, assume direct assignment
		return varName
	}
}

// Generate assignment code for nullable embeds after scanning
func (v QueryValue) NullableEmbedAssignments() []string {
	var assignments []string
	if v.Struct == nil {
		return assignments
	}
	for _, f := range v.Struct.Fields {
		if len(f.EmbedFields) > 0 && f.EmbedIsNullable {
			// Find primary key fields
			var primaryKeyChecks []string
			if len(f.EmbedPrimaryKeys) > 0 {
				// Use actual primary key fields
				for _, pk := range f.EmbedPrimaryKeys {
					checkVar := fmt.Sprintf("%sEmbed%s%s", v.Name, f.Name, pk.Name)
					// Check if the nullable primary key is valid
					primaryKeyChecks = append(primaryKeyChecks, fmt.Sprintf("%s.Valid", checkVar))
				}
			} else if len(f.EmbedFields) > 0 {
				// Fallback: if no primary key is found, panic or error
				panic(fmt.Sprintf("No primary key found for embedded table %s. Primary key information is required for nullable LEFT JOIN handling.", f.Name))
			}
			
			if len(primaryKeyChecks) > 0 {
				// Start the if statement with primary key checks
				checkCondition := strings.Join(primaryKeyChecks, " || ")
				assignments = append(assignments, fmt.Sprintf("if %s {", checkCondition))
				assignments = append(assignments, fmt.Sprintf("    %s.%s.Valid = true", v.Name, f.Name))
				
				// Build the struct initialization
				structType := f.Type
				if strings.HasPrefix(structType, "sql.Null[") && strings.HasSuffix(structType, "]") {
					structType = structType[9:len(structType)-1] // Extract type from sql.Null[Type]
				}
				
				assignment := fmt.Sprintf("    %s.%s.V = %s{", v.Name, f.Name, structType)
				assignments = append(assignments, assignment)
				
				// Add field assignments, extracting values from nullable types
				for _, embed := range f.EmbedFields {
					tempVar := fmt.Sprintf("%sEmbed%s%s", v.Name, f.Name, embed.Name)
					value := extractNullableValue(tempVar, embed.Type)
					assignments = append(assignments, fmt.Sprintf("        %s: %s,", embed.Name, value))
				}
				
				assignments = append(assignments, "    }")
				assignments = append(assignments, "}")
			}
		}
	}
	return assignments
}

func (v QueryValue) ScanWithOptions(emitEmbedNullableLeftJoin bool) string {
	var out []string
	if v.Struct == nil {
		if strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array(&"+v.Name+")")
		} else {
			out = append(out, "&"+v.Name)
		}
	} else {
		for _, f := range v.Struct.Fields {

			// Handle nullable embedded fields differently
			if len(f.EmbedFields) > 0 && f.EmbedIsNullable && emitEmbedNullableLeftJoin {
				// For nullable embeds, we need to scan into temporary fields
				for _, embed := range f.EmbedFields {
					tempName := v.Name + "Embed" + f.Name + embed.Name
					if strings.HasPrefix(embed.Type, "[]") && embed.Type != "[]byte" && !v.SQLDriver.IsPGX() {
						out = append(out, "pq.Array(&"+tempName+")")
					} else {
						out = append(out, "&"+tempName)
					}
				}
				continue
			}

			// append any embedded fields
			if len(f.EmbedFields) > 0 {
				for _, embed := range f.EmbedFields {
					if strings.HasPrefix(embed.Type, "[]") && embed.Type != "[]byte" && !v.SQLDriver.IsPGX() {
						out = append(out, "pq.Array(&"+v.Name+"."+f.Name+"."+embed.Name+")")
					} else {
						out = append(out, "&"+v.Name+"."+f.Name+"."+embed.Name)
					}
				}
				continue
			}

			if strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array(&"+v.Name+"."+f.Name+")")
			} else {
				out = append(out, "&"+v.Name+"."+f.Name)
			}
		}
	}
	if len(out) <= 3 {
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

// Deprecated: This method does not respect the Emit field set on the
// QueryValue. It's used by the go-sql-driver-mysql/copyfromCopy.tmpl and should
// not be used other places.
func (v QueryValue) CopyFromMySQLFields() []Field {
	// fmt.Printf("%#v\n", v)
	if v.Struct != nil {
		return v.Struct.Fields
	}
	return []Field{
		{
			Name:   v.Name,
			DBName: v.DBName,
			Type:   v.Typ,
		},
	}
}

func (v QueryValue) VariableForField(f Field) string {
	if !v.IsStruct() {
		return v.Name
	}
	if !v.EmitStruct() {
		return toLowerCase(f.Name)
	}
	return v.Name + "." + f.Name
}

// A struct used to generate methods and fields on the Queries struct
type Query struct {
	Cmd          string
	Comments     []string
	MethodName   string
	FieldName    string
	ConstantName string
	SQL          string
	SourceName   string
	Ret          QueryValue
	Arg          QueryValue
	// Used for :copyfrom
	Table *plugin.Identifier
}

func (q Query) hasRetType() bool {
	scanned := q.Cmd == metadata.CmdOne || q.Cmd == metadata.CmdMany ||
		q.Cmd == metadata.CmdBatchMany || q.Cmd == metadata.CmdBatchOne
	return scanned && !q.Ret.isEmpty()
}

func (q Query) TableIdentifierAsGoSlice() string {
	escapedNames := make([]string, 0, 3)
	for _, p := range []string{q.Table.Catalog, q.Table.Schema, q.Table.Name} {
		if p != "" {
			escapedNames = append(escapedNames, fmt.Sprintf("%q", p))
		}
	}
	return "[]string{" + strings.Join(escapedNames, ", ") + "}"
}

func (q Query) TableIdentifierForMySQL() string {
	escapedNames := make([]string, 0, 3)
	for _, p := range []string{q.Table.Catalog, q.Table.Schema, q.Table.Name} {
		if p != "" {
			escapedNames = append(escapedNames, fmt.Sprintf("`%s`", p))
		}
	}
	return strings.Join(escapedNames, ".")
}
