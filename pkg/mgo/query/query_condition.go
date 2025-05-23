// Package query is a library of custom condition queries, support for complex conditional paging queries.
package query

import (
	"fmt"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// Eq equal
	Eq       = "eq"
	eqSymbol = "="
	// Neq not equal
	Neq       = "neq"
	neqSymbol = "!="
	// Gt greater than
	Gt       = "gt"
	gtSymbol = ">"
	// Gte greater than or equal
	Gte       = "gte"
	gteSymbol = ">="
	// Lt less than
	Lt       = "lt"
	ltSymbol = "<"
	// Lte less than or equal
	Lte       = "lte"
	lteSymbol = "<="
	// Like fuzzy lookup
	Like = "like"
	// In include
	In = "in"
	// NotIn exclude
	NotIn = "nin"

	// AND logic and
	AND        string = "and" //nolint
	andSymbol1        = "&"
	andSymbol2        = "&&"
	// OR logic or
	OR        string = "or" //nolint
	orSymbol1        = "|"
	orSymbol2        = "||"

	allLogicAnd = 1
	allLogicOr  = 2
)

var expMap = map[string]string{
	Eq:        eqSymbol,
	eqSymbol:  eqSymbol,
	Neq:       neqSymbol,
	neqSymbol: neqSymbol,
	Gt:        gtSymbol,
	gtSymbol:  gtSymbol,
	Gte:       gteSymbol,
	gteSymbol: gteSymbol,
	Lt:        ltSymbol,
	ltSymbol:  ltSymbol,
	Lte:       lteSymbol,
	lteSymbol: lteSymbol,
	Like:      Like,
	In:        In,
	NotIn:     NotIn,
	"notin":   NotIn,
	"not in":  NotIn,
}

var logicMap = map[string]string{
	AND:        andSymbol1,
	andSymbol1: andSymbol1,
	andSymbol2: andSymbol1,
	OR:         orSymbol1,
	orSymbol1:  orSymbol1,
	orSymbol2:  orSymbol1,
}

// ---------------------------------------------------------------------------

type rulerOptions struct {
	whitelistNames map[string]bool
	validateFn     func(columns []Column) error
}

// RulerOption set the parameters of ruler options
type RulerOption func(*rulerOptions)

func (o *rulerOptions) apply(opts ...RulerOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithWhitelistNames set white list names of columns
func WithWhitelistNames(whitelistNames map[string]bool) RulerOption {
	return func(o *rulerOptions) {
		o.whitelistNames = whitelistNames
	}
}

// WithValidateFn set validate function of columns
func WithValidateFn(fn func(columns []Column) error) RulerOption {
	return func(o *rulerOptions) {
		o.validateFn = fn
	}
}

// -----------------------------------------------------------------------------

// Params query parameters
type Params struct {
	Page  int    `json:"page" form:"page" binding:"gte=0"`
	Limit int    `json:"limit" form:"limit" binding:"gte=1"`
	Sort  string `json:"sort,omitempty" form:"sort" binding:""`

	Columns []Column `json:"columns,omitempty" form:"columns"` // not required

	// Deprecated: use Limit instead in sponge version v1.8.6, will remove in the future
	Size int `json:"size" form:"size"`
}

// Column query info
type Column struct {
	Name  string      `json:"name" form:"name"`   // column name
	Exp   string      `json:"exp" form:"exp"`     // expressions, default value is "=", support =, !=, >, >=, <, <=, like, in
	Value interface{} `json:"value" form:"value"` // column value
	Logic string      `json:"logic" form:"logic"` // logical type, defaults to and when the value is null, with &(and), ||(or)
}

func (c *Column) checkName(whitelists map[string]bool) error {
	if c.Name == "" || (whitelists != nil && !whitelists[c.Name]) {
		return fmt.Errorf("field name '%s' is not allowed", c.Name)
	}
	return nil
}

func (c *Column) checkValid() error {
	if c.Name == "" {
		return fmt.Errorf("field 'name' cannot be empty")
	}
	if c.Value == nil {
		return fmt.Errorf("field 'value' cannot be nil")
	}
	return nil
}

func (c *Column) convertLogic() error {
	if c.Logic == "" {
		c.Logic = AND
	}
	if v, ok := logicMap[strings.ToLower(c.Logic)]; ok { //nolint
		c.Logic = v
		return nil
	}
	return fmt.Errorf("unknown logic type '%s'", c.Logic)
}

// converting ExpType to sql expressions and LogicType to sql using characters
func (c *Column) convert() error {
	if err := c.checkValid(); err != nil {
		return err
	}

	if oid, ok := isObjectID(c.Value); ok {
		c.Value = oid

		if c.Name == "id" {
			c.Name = "_id" // force to "_id"
		} else if strings.HasSuffix(c.Name, ":oid") {
			c.Name = strings.TrimSuffix(c.Name, ":oid")
		}
	}

	if c.Exp == "" {
		c.Exp = Eq
	}
	if v, ok := expMap[strings.ToLower(c.Exp)]; ok { //nolint
		c.Exp = v
		switch c.Exp {
		//case eqSymbol:
		case neqSymbol:
			c.Value = bson.M{"$ne": c.Value}
		case gtSymbol:
			c.Value = bson.M{"$gt": c.Value}
		case gteSymbol:
			c.Value = bson.M{"$gte": c.Value}
		case ltSymbol:
			c.Value = bson.M{"$lt": c.Value}
		case lteSymbol:
			c.Value = bson.M{"$lte": c.Value}
		case Like:
			escapedValue := regexp.QuoteMeta(fmt.Sprintf("%v", c.Value))
			c.Value = bson.M{"$regex": escapedValue, "$options": "i"}
		case In, NotIn:
			val, ok2 := c.Value.(string)
			if !ok2 {
				return fmt.Errorf("invalid value type '%s'", c.Value)
			}
			values := []interface{}{}
			ss := strings.Split(val, ",")
			for _, s := range ss {
				values = append(values, s)
			}
			c.Value = bson.M{"$" + c.Exp: values}
		}
	} else {
		return fmt.Errorf("unsported exp type '%s'", c.Exp)
	}

	return c.convertLogic()
}

// ConvertToPage converted to page
func (p *Params) ConvertToPage() (sort bson.D, limit int, skip int) { //nolint
	page := NewPage(p.Page, p.Limit, p.Sort)
	sort = page.sort
	limit = page.limit
	skip = page.page * page.limit
	return //nolint
}

// ConvertToMongoFilter conversion to mongo-compliant parameters based on the Columns parameter
// ignore the logical type of the last column, whether it is a one-column or multi-column query
func (p *Params) ConvertToMongoFilter(opts ...RulerOption) (bson.M, error) {
	o := rulerOptions{}
	o.apply(opts...)
	if o.validateFn != nil {
		err := o.validateFn(p.Columns)
		if err != nil {
			return nil, err
		}
	}

	filter := bson.M{}
	l := len(p.Columns)
	switch l {
	case 0:
		return bson.M{}, nil

	case 1: // l == 1
		err := p.Columns[0].checkName(o.whitelistNames)
		if err != nil {
			return nil, err
		}
		err = p.Columns[0].convert()
		if err != nil {
			return nil, err
		}
		filter[p.Columns[0].Name] = p.Columns[0].Value
		return filter, nil

	case 2: // l == 2
		err := p.Columns[0].checkName(o.whitelistNames)
		if err != nil {
			return nil, err
		}
		err = p.Columns[1].checkName(o.whitelistNames)
		if err != nil {
			return nil, err
		}
		err = p.Columns[0].convert()
		if err != nil {
			return nil, err
		}
		err = p.Columns[1].convert()
		if err != nil {
			return nil, err
		}
		if p.Columns[0].Logic == andSymbol1 {
			filter = bson.M{"$and": []bson.M{
				{p.Columns[0].Name: p.Columns[0].Value},
				{p.Columns[1].Name: p.Columns[1].Value}}}
		} else {
			filter = bson.M{"$or": []bson.M{
				{p.Columns[0].Name: p.Columns[0].Value},
				{p.Columns[1].Name: p.Columns[1].Value}}}
		}
		return filter, nil

	default: // l >=3
		return p.convertMultiColumns(o.whitelistNames)
	}
}

func (p *Params) convertMultiColumns(whitelistNames map[string]bool) (bson.M, error) {
	filter := bson.M{}
	logicType, groupIndexes, err := checkSameLogic(p.Columns)
	if err != nil {
		return nil, err
	}
	if logicType == allLogicAnd {
		for _, column := range p.Columns {
			err = column.checkName(whitelistNames)
			if err != nil {
				return nil, err
			}

			err = column.convert()
			if err != nil {
				return nil, err
			}
			if v, ok := filter["$and"]; !ok {
				filter["$and"] = []bson.M{{column.Name: column.Value}}
			} else {
				if cols, ok1 := v.([]bson.M); ok1 {
					cols = append(cols, bson.M{column.Name: column.Value})
					filter["$and"] = cols
				}
			}
		}
		return filter, nil
	} else if logicType == allLogicOr {
		for _, column := range p.Columns {
			err = column.convert()
			if err != nil {
				return nil, err
			}
			if v, ok := filter["$or"]; !ok {
				filter["$or"] = []bson.M{{column.Name: column.Value}}
			} else {
				if cols, ok1 := v.([]bson.M); ok1 {
					cols = append(cols, bson.M{column.Name: column.Value})
					filter["$or"] = cols
				}
			}
		}
		return filter, nil
	}
	orConditions := []bson.M{}
	for _, indexes := range groupIndexes {
		if len(indexes) == 1 {
			column := p.Columns[indexes[0]]
			err := column.convert()
			if err != nil {
				return nil, err
			}
			orConditions = append(orConditions, bson.M{column.Name: column.Value})
		} else {
			andConditions := []bson.M{}
			for _, index := range indexes {
				column := p.Columns[index]
				err := column.convert()
				if err != nil {
					return nil, err
				}
				andConditions = append(andConditions, bson.M{column.Name: column.Value})
			}
			orConditions = append(orConditions, bson.M{"$and": andConditions})
		}
	}
	filter["$or"] = orConditions

	return filter, nil
}

func isObjectID(v interface{}) (primitive.ObjectID, bool) {
	if str, ok := v.(string); ok && len(str) == 24 {
		value, err := primitive.ObjectIDFromHex(str)
		if err == nil {
			return value, true
		}
	}
	return [12]byte{}, false
}

func checkSameLogic(columns []Column) (int, [][]int, error) {
	orIndexes := []int{}
	l := len(columns)
	for i, column := range columns {
		if i == l-1 { // ignore the logical type of the last column
			break
		}
		err := column.convertLogic()
		if err != nil {
			return 0, nil, err
		}
		if column.Logic == orSymbol1 {
			orIndexes = append(orIndexes, i)
		}
	}

	if len(orIndexes) == 0 {
		return allLogicAnd, nil, nil
	} else if len(orIndexes) == l-1 {
		return allLogicOr, nil, nil
	}
	// mix and or
	groupIndexes := groupingIndex(l, orIndexes)

	return 0, groupIndexes, nil
}

func groupingIndex(l int, orIndexes []int) [][]int {
	groupIndexes := [][]int{}
	lastIndex := 0
	for _, index := range orIndexes {
		group := []int{}
		for i := lastIndex; i <= index; i++ {
			group = append(group, i)
		}
		groupIndexes = append(groupIndexes, group)
		if lastIndex == index {
			lastIndex++
		} else {
			lastIndex = index
		}
	}
	group := []int{}
	for i := lastIndex + 1; i < l; i++ {
		group = append(group, i)
	}
	groupIndexes = append(groupIndexes, group)
	return groupIndexes
}

// Conditions query conditions
type Conditions struct {
	Columns []Column `json:"columns" form:"columns" binding:"min=1"` // columns info
}

// CheckValid check valid
func (c *Conditions) CheckValid() error {
	if len(c.Columns) == 0 {
		return fmt.Errorf("field 'columns' cannot be empty")
	}

	for _, column := range c.Columns {
		err := column.checkValid()
		if err != nil {
			return err
		}
		if column.Exp != "" {
			if _, ok := expMap[column.Exp]; !ok {
				return fmt.Errorf("unknown exp type '%s'", column.Exp)
			}
		}
		if column.Logic != "" {
			if _, ok := logicMap[column.Logic]; !ok {
				return fmt.Errorf("unknown logic type '%s'", column.Logic)
			}
		}
	}

	return nil
}

// ConvertToMongo conversion to mongo-compliant parameters based on the Columns parameter
// ignore the logical type of the last column, whether it is a one-column or multi-column query
func (c *Conditions) ConvertToMongo(opts ...RulerOption) (bson.M, error) {
	p := &Params{Columns: c.Columns}
	return p.ConvertToMongoFilter(opts...)
}
