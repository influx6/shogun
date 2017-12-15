// Package context is built out of my desire to understand the http context
// library and as an experiement in such a library works.
package context

import (
	gcontext "context"
	"sync"
	"time"
)

// Fields defines a map of key:value pairs.
type Fields map[interface{}]interface{}

// Getter defines a series of Get methods for which values will be retrieved with.
type Getter interface {
	Get(key interface{}) (interface{}, bool)
	GetInt(key interface{}) (int, bool)
	GetBool(key interface{}) (bool, bool)
	GetInt8(key interface{}) (int8, bool)
	GetInt16(key interface{}) (int16, bool)
	GetInt32(key interface{}) (int32, bool)
	GetInt64(key interface{}) (int64, bool)
	GetString(key interface{}) (string, bool)
	GetFloat32(key interface{}) (float32, bool)
	GetFloat64(key interface{}) (float64, bool)
	GetDuration(key interface{}) (time.Duration, bool)
}

// ValueBag defines a context for holding values to be shared across processes..
type ValueBag interface {
	Getter

	// Set adds a key-value pair into the bag.
	Set(key, value interface{})

	// WithValue returns a new context then adds the key and value pair into the
	// context's store.
	WithValue(key interface{}, value interface{}) ValueBag
}

// IsExpired returns true/false whether the provided CancelContext has expired.
func IsExpired(c CancelContext) bool {
	select {
	case <-c.Done():
		return true
	case <-time.After(5 * time.Millisecond):
		return false
	}
}

// Deadline exposes a single method to return expected deadline for context.
type Deadline interface {
	Deadline() (time.Time, bool)
}

// CancelContext defines a type which provides Done signal for cancelling operations.
type CancelContext interface {
	Deadline
	Done() <-chan struct{}
}

// Context defines a type which holds a cancel signal and contains
// a bag of values.
type Context interface {
	CancelContext
	Bag() ValueBag
}

// CancelableContext defines a type which provides Done signal for cancelling operations.
type CancelableContext interface {
	Context
	Cancel()
}

// CnclContext defines a struct to implement the CancelContext.
type CnclContext struct {
	close chan struct{}
	once  sync.Once
	bag   ValueBag
}

// MakeGoogleContextFrom returns a goole context package instance by using the CancelContext
// to cancel the returned context.
func MakeGoogleContextFrom(ctx CancelContext) gcontext.Context {
	cmx, canceler := gcontext.WithCancel(gcontext.Background())
	go func() {
		<-ctx.Done()
		canceler()
	}()
	return cmx
}

// New returns a new instance of a CancelableContext with ValueBag set.
func New() CancelableContext {
	return &CnclContext{close: make(chan struct{}), bag: NewValueBag()}
}

// WithTimeout returns a new Context made from provided duration.
func WithTimeout(bag ValueBag, d time.Duration) Context {
	return NewExpiringCnclContext(nil, d, bag)
}

// NewCnclContext returns a new instance of the CnclContext.
func NewCnclContext(bag ValueBag) *CnclContext {
	return &CnclContext{close: make(chan struct{}), bag: bag}
}

// Bag returns an associated ValueBag for this instance.
func (cn *CnclContext) Bag() ValueBag {
	if cn.bag == nil {
		cn.bag = NewValueBag()
	}

	return cn.bag
}

// Cancel closes the internal channel of the contxt
func (cn *CnclContext) Cancel() {
	cn.once.Do(func() {
		close(cn.close)
	})
}

// Deadline returns giving time when context is expected to be canceled.
func (cn *CnclContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

// Done returns a channel to signal ending of op.
// It implements the CancelContext.
func (cn *CnclContext) Done() <-chan struct{} {
	return cn.close
}

// ExpiringCnclContext defines a struct to implement the CancelContext.
type ExpiringCnclContext struct {
	close    chan struct{}
	action   func()
	once     sync.Once
	duration time.Duration
	deadline time.Time
	bag      ValueBag
}

// NewExpiringCnclContext returns a new instance of the CnclContext.
func NewExpiringCnclContext(action func(), timeout time.Duration, bag ValueBag) *ExpiringCnclContext {
	exp := &ExpiringCnclContext{close: make(chan struct{}), action: action, bag: bag, duration: timeout, deadline: time.Now().Add(timeout)}
	go exp.monitor()
	return exp
}

// Deadline returns giving time when context is expected to be canceled.
func (cn *ExpiringCnclContext) Deadline() (time.Time, bool) {
	return cn.deadline, true
}

// Cancel closes the internal channel of the contxt
func (cn *ExpiringCnclContext) Cancel() {
	cn.once.Do(func() {
		// cn.mu.Lock()
		// defer cn.mu.Unlock()

		close(cn.close)
		cn.bag = nil
		if cn.action != nil {
			cn.action()
		}
	})
}

// Bag returns an associated ValueBag for this instance.
func (cn *ExpiringCnclContext) Bag() ValueBag {
	// cn.mu.Lock()
	// defer cn.mu.Unlock()

	if cn.bag == nil {
		cn.bag = NewValueBag()
	}

	return cn.bag
}

// Done returns a channel to signal ending of op.
// It implements the CancelContext.
func (cn *ExpiringCnclContext) Done() <-chan struct{} {
	return cn.close
}

func (cn *ExpiringCnclContext) monitor() {
	<-time.After(cn.duration)
	cn.Cancel()
}

//================================================================================

// context defines a struct for bundling a context against specific
// use cases with a explicitly set duration which clears all its internal
// data after the giving period.
type context struct {
	ml     sync.Mutex
	fields *Pair
}

// ValueBagFromAny adds giving key-value pairs into the bag.
func ValueBagFromAny(fields map[interface{}]interface{}) ValueBag {
	initial := (*Pair)(nil)

	for key, val := range fields {
		initial = initial.Append(key, val)
	}

	return &context{fields: initial}
}

// ValueBagFrom adds giving key-value pairs into the bag.
func ValueBagFrom(fields map[string]interface{}) ValueBag {
	initial := (*Pair)(nil)

	for key, val := range fields {
		initial = initial.Append(key, val)
	}

	return &context{fields: initial}
}

// NewValueBag returns a new context object that meets the Context interface.
func NewValueBag() ValueBag {
	cl := context{
		fields: (*Pair)(nil),
	}

	return &cl
}

// Set adds given value into context.
func (c *context) Set(key, value interface{}) {
	c.ml.Lock()
	defer c.ml.Unlock()
	c.fields = Append(c.fields, key, value)
}

// WithValue returns a new context based on the previos one.
func (c *context) WithValue(key, value interface{}) ValueBag {
	c.ml.Lock()
	defer c.ml.Unlock()
	child := &context{
		fields: Append(c.fields, key, value),
	}

	return child
}

// GetDuration returns the value for the necessary key within the context.
func (c *context) GetDuration(key interface{}) (item time.Duration, found bool) {
	c.ml.Lock()
	defer c.ml.Unlock()
	item, found = c.fields.GetDuration(key)
	return
}

// Get returns the value for the necessary key within the context.
func (c *context) Get(key interface{}) (item interface{}, found bool) {
	c.ml.Lock()
	defer c.ml.Unlock()
	item, found = c.fields.Get(key)
	return
}

// GetBool returns the value type value of a key if it exists.
func (c *context) GetBool(key interface{}) (bool, bool) {
	return c.fields.GetBool(key)
}

// GetFloat64 returns the value type value of a key if it exists.
func (c *context) GetFloat64(key interface{}) (float64, bool) {
	return c.fields.GetFloat64(key)
}

// GetFloat32 returns the value type value of a key if it exists.
func (c *context) GetFloat32(key interface{}) (float32, bool) {
	return c.fields.GetFloat32(key)
}

// GetInt8 returns the value type value of a key if it exists.
func (c *context) GetInt8(key interface{}) (int8, bool) {
	return c.fields.GetInt8(key)
}

// GetInt16 returns the value type value of a key if it exists.
func (c *context) GetInt16(key interface{}) (int16, bool) {
	return c.fields.GetInt16(key)
}

// GetInt64 returns the value type value of a key if it exists.
func (c *context) GetInt64(key interface{}) (int64, bool) {
	return c.fields.GetInt64(key)
}

// GetInt32 returns the value type value of a key if it exists.
func (c *context) GetInt32(key interface{}) (int32, bool) {
	return c.fields.GetInt32(key)
}

// GetInt returns the value type value of a key if it exists.
func (c *context) GetInt(key interface{}) (int, bool) {
	return c.fields.GetInt(key)
}

// GetString returns the value type value of a key if it exists.
func (c *context) GetString(key interface{}) (string, bool) {
	return c.fields.GetString(key)
}

// Deadline returns giving time when context is expected to be canceled.
func (c *context) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

//==============================================================================

// GoogleContext implements a decorator for googles context package.
type GoogleContext struct {
	gcontext.Context
}

// FromContext returns a new context object that meets the Context interface.
func FromContext(ctx gcontext.Context) *GoogleContext {
	var gc GoogleContext
	gc.Context = ctx
	return &gc
}

// GetDuration returns the giving value for the provided key if it exists else nil.
func (g *GoogleContext) GetDuration(key interface{}) (time.Duration, bool) {
	val := g.Context.Value(key)
	if val == nil {
		return 0, false
	}

	if dval, ok := val.(time.Duration); ok {
		return dval, true
	}

	if dval, ok := val.(int64); ok {
		return time.Duration(dval), true
	}

	if sval, ok := val.(string); ok {
		if dur, err := time.ParseDuration(sval); err == nil {
			return dur, true
		}
	}

	return 0, false
}

// Get returns the giving value for the provided key if it exists else nil.
func (g *GoogleContext) Get(key interface{}) (interface{}, bool) {
	val := g.Context.Value(key)
	if val == nil {
		return val, false
	}

	return val, true
}

// GetBool returns the value type value of a key if it exists.
func (g *GoogleContext) GetBool(key interface{}) (bool, bool) {
	val, found := g.Get(key)
	if !found {
		return false, false
	}

	value, ok := val.(bool)
	return value, ok
}

// GetFloat64 returns the value type value of a key if it exists.
func (g *GoogleContext) GetFloat64(key interface{}) (float64, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float64)
	return value, ok
}

// GetFloat32 returns the value type value of a key if it exists.
func (g *GoogleContext) GetFloat32(key interface{}) (float32, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float32)
	return value, ok
}

// GetInt8 returns the value type value of a key if it exists.
func (g *GoogleContext) GetInt8(key interface{}) (int8, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int8)
	return value, ok
}

// GetInt16 returns the value type value of a key if it exists.
func (g *GoogleContext) GetInt16(key interface{}) (int16, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int16)
	return value, ok
}

// GetInt64 returns the value type value of a key if it exists.
func (g *GoogleContext) GetInt64(key interface{}) (int64, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int64)
	return value, ok
}

// GetInt32 returns the value type value of a key if it exists.
func (g *GoogleContext) GetInt32(key interface{}) (int32, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int32)
	return value, ok
}

// GetInt returns the value type value of a key if it exists.
func (g *GoogleContext) GetInt(key interface{}) (int, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int)
	return value, ok
}

// GetString returns the value type value of a key if it exists.
func (g *GoogleContext) GetString(key interface{}) (string, bool) {
	val, found := g.Get(key)
	if !found {
		return "", false
	}

	value, ok := val.(string)
	return value, ok
}

//==============================================================================

// Pair defines a struct for storing a linked pair of key and values.
type Pair struct {
	prev  *Pair
	key   interface{}
	value interface{}
}

// NewPair returns a a key-value pair chain for setting fields.
func NewPair(key, value interface{}) *Pair {
	return &Pair{
		key:   key,
		value: value,
	}
}

// Append returns a new Pair with the giving key and with the provded Pair set as
// it's previous link.
func Append(p *Pair, key, value interface{}) *Pair {
	return p.Append(key, value)
}

// Fields returns all internal pair data as a map.
func (p *Pair) Fields() Fields {
	var f Fields

	if p.prev == nil {
		f = make(Fields)
		f[p.key] = p.value
		return f
	}

	f = p.prev.Fields()

	if p.key != "" {
		f[p.key] = p.value
	}

	return f
}

// Append returns a new pair with the giving key and value and its previous
// set to this pair.
func (p *Pair) Append(key, val interface{}) *Pair {
	return &Pair{
		prev:  p,
		key:   key,
		value: val,
	}
}

// RemoveAll sets all key-value pairs to nil for all connected pair, till it reaches
// the root.
func (p *Pair) RemoveAll() {
	p.key = nil
	p.value = nil

	if p.prev != nil {
		p.prev.RemoveAll()
	}
}

// Root returns the root Pair in the chain which links all pairs together.
func (p *Pair) Root() *Pair {
	if p.prev == nil {
		return p
	}

	return p.prev.Root()
}

// GetDuration returns the duration value of a key if it exists.
func (p *Pair) GetDuration(key interface{}) (time.Duration, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	if dval, ok := val.(time.Duration); ok {
		return dval, true
	}

	if dval, ok := val.(int64); ok {
		return time.Duration(dval), true
	}

	if sval, ok := val.(string); ok {
		if dur, err := time.ParseDuration(sval); err == nil {
			return dur, true
		}
	}

	return 0, false
}

// GetBool returns the bool value of a key if it exists.
func (p *Pair) GetBool(key interface{}) (bool, bool) {
	val, found := p.Get(key)
	if !found {
		return false, false
	}

	value, ok := val.(bool)
	return value, ok
}

// GetFloat64 returns the float64 value of a key if it exists.
func (p *Pair) GetFloat64(key interface{}) (float64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float64)
	return value, ok
}

// GetFloat32 returns the float32 value of a key if it exists.
func (p *Pair) GetFloat32(key interface{}) (float32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float32)
	return value, ok
}

// GetInt8 returns the int8 value of a key if it exists.
func (p *Pair) GetInt8(key interface{}) (int8, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int8)
	return value, ok
}

// GetInt16 returns the int16 value of a key if it exists.
func (p *Pair) GetInt16(key interface{}) (int16, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int16)
	return value, ok
}

// GetInt64 returns the value type value of a key if it exists.
func (p *Pair) GetInt64(key interface{}) (int64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int64)
	return value, ok
}

// GetInt32 returns the value type value of a key if it exists.
func (p *Pair) GetInt32(key interface{}) (int32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int32)
	return value, ok
}

// GetInt returns the value type value of a key if it exists.
func (p *Pair) GetInt(key interface{}) (int, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int)
	return value, ok
}

// GetString returns the value type value of a key if it exists.
func (p *Pair) GetString(key interface{}) (string, bool) {
	val, found := p.Get(key)
	if !found {
		return "", false
	}

	value, ok := val.(string)
	return value, ok
}

// Get returns the value of a key if it exists.
func (p *Pair) Get(key interface{}) (value interface{}, found bool) {
	if p == nil {
		return
	}

	if p.key == key {
		return p.value, true
	}

	if p.prev == nil {
		return
	}

	return p.prev.Get(key)
}
