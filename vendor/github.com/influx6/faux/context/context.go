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
}

// ValueBagContext defines a context for holding values to be shared across processes..
type ValueBagContext interface {
	Getter

	// Set adds a key and value pair into the context store.
	Set(key interface{}, value interface{})

	// WithValue returns a new context then adds the key and value pair into the
	// context's store.
	WithValue(key interface{}, value interface{}) ValueBagContext
}

//==============================================================================

// CancelContext defines a type which provides Done signal for cancelling operations.
type CancelContext interface {
	Done() <-chan struct{}
}

// CancelableContext defines a type which provides Done signal for cancelling operations.
type CancelableContext interface {
	Done() <-chan struct{}
	Cancel()
}

// CnclContext defines a struct to implement the CancelContext.
type CnclContext struct {
	close chan struct{}
	once  sync.Once
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

// NewCnclContext returns a new instance of the CnclContext.
func NewCnclContext() *CnclContext {
	return &CnclContext{close: make(chan struct{})}
}

// Cancel closes the internal channel of the contxt
func (cn *CnclContext) Cancel() {
	cn.once.Do(func() {
		close(cn.close)
	})
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
}

// NewExpiringCnclContext returns a new instance of the CnclContext.
func NewExpiringCnclContext(action func(), timeout time.Duration) *ExpiringCnclContext {
	exp := &ExpiringCnclContext{close: make(chan struct{}), action: action}
	go exp.monitor()
	return exp
}

// Cancel closes the internal channel of the contxt
func (cn *ExpiringCnclContext) Cancel() {
	cn.once.Do(func() {
		close(cn.close)
		if cn.action != nil {
			cn.action()
		}
	})
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

//==============================================================================

// nilPair defines a nil starting pair.
var nilPair = (*Pair)(nil)

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

// GetBool collects the string value of a key if it exists.
func (p *Pair) GetBool(key interface{}) (bool, bool) {
	val, found := p.Get(key)
	if !found {
		return false, false
	}

	value, ok := val.(bool)
	return value, ok
}

// GetFloat64 collects the string value of a key if it exists.
func (p *Pair) GetFloat64(key interface{}) (float64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float64)
	return value, ok
}

// GetFloat32 collects the string value of a key if it exists.
func (p *Pair) GetFloat32(key interface{}) (float32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float32)
	return value, ok
}

// GetInt8 collects the string value of a key if it exists.
func (p *Pair) GetInt8(key interface{}) (int8, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int8)
	return value, ok
}

// GetInt16 collects the string value of a key if it exists.
func (p *Pair) GetInt16(key interface{}) (int16, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int16)
	return value, ok
}

// GetInt64 collects the string value of a key if it exists.
func (p *Pair) GetInt64(key interface{}) (int64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int64)
	return value, ok
}

// GetInt32 collects the string value of a key if it exists.
func (p *Pair) GetInt32(key interface{}) (int32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int32)
	return value, ok
}

// GetInt collects the string value of a key if it exists.
func (p *Pair) GetInt(key interface{}) (int, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int)
	return value, ok
}

// GetString collects the string value of a key if it exists.
func (p *Pair) GetString(key interface{}) (string, bool) {
	val, found := p.Get(key)
	if !found {
		return "", false
	}

	value, ok := val.(string)
	return value, ok
}

// Get collects the value of a key if it exists.
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

// Get returns the giving value for the provided key if it exists else nil.
func (g *GoogleContext) Get(key interface{}) (interface{}, bool) {
	val := g.Context.Value(key)
	if val == nil {
		return val, false
	}

	return val, true
}

// GetBool collects the string value of a key if it exists.
func (g *GoogleContext) GetBool(key interface{}) (bool, bool) {
	val, found := g.Get(key)
	if !found {
		return false, false
	}

	value, ok := val.(bool)
	return value, ok
}

// GetFloat64 collects the string value of a key if it exists.
func (g *GoogleContext) GetFloat64(key interface{}) (float64, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float64)
	return value, ok
}

// GetFloat32 collects the string value of a key if it exists.
func (g *GoogleContext) GetFloat32(key interface{}) (float32, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float32)
	return value, ok
}

// GetInt8 collects the string value of a key if it exists.
func (g *GoogleContext) GetInt8(key interface{}) (int8, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int8)
	return value, ok
}

// GetInt16 collects the string value of a key if it exists.
func (g *GoogleContext) GetInt16(key interface{}) (int16, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int16)
	return value, ok
}

// GetInt64 collects the string value of a key if it exists.
func (g *GoogleContext) GetInt64(key interface{}) (int64, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int64)
	return value, ok
}

// GetInt32 collects the string value of a key if it exists.
func (g *GoogleContext) GetInt32(key interface{}) (int32, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int32)
	return value, ok
}

// GetInt collects the string value of a key if it exists.
func (g *GoogleContext) GetInt(key interface{}) (int, bool) {
	val, found := g.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int)
	return value, ok
}

// GetString collects the string value of a key if it exists.
func (g *GoogleContext) GetString(key interface{}) (string, bool) {
	val, found := g.Get(key)
	if !found {
		return "", false
	}

	value, ok := val.(string)
	return value, ok
}

//================================================================================

// context defines a struct for bundling a context against specific
// use cases with a explicitly set duration which clears all its internal
// data after the giving period.
type context struct {
	mx     sync.Mutex
	fields *Pair
}

// ExpiringValueBag returns a ValueBagContext which contexts will be deleted once
// the provided duration has finished it's
func ExpiringValueBag(dur time.Duration) ValueBagContext {
	bag := &context{
		fields: nilPair,
	}

	NewExpiringCnclContext(func() {
		bag.mx.Lock()
		defer bag.mx.Unlock()
		bag.fields = nilPair
	}, dur)

	return bag
}

// ValueBag returns a new context object that meets the Context interface.
func ValueBag() ValueBagContext {
	cl := context{
		fields: nilPair,
	}

	return &cl
}

// WithValue returns a new context based on the previos one.
func (c *context) WithValue(key, value interface{}) ValueBagContext {
	c.mx.Lock()
	fields := Append(c.fields, key, value)
	c.mx.Unlock()

	child := &context{
		fields: fields,
	}

	return child
}

// Set adds the giving value using the given key into the map.
func (c *context) Set(key, val interface{}) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.fields = Append(c.fields, key, val)
}

// Get returns the value for the necessary key within the context.
func (c *context) Get(key interface{}) (item interface{}, found bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	item, found = c.fields.Get(key)
	return
}

// GetBool collects the string value of a key if it exists.
func (c *context) GetBool(key interface{}) (bool, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetBool(key)
}

// GetFloat64 collects the string value of a key if it exists.
func (c *context) GetFloat64(key interface{}) (float64, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetFloat64(key)
}

// GetFloat32 collects the string value of a key if it exists.
func (c *context) GetFloat32(key interface{}) (float32, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetFloat32(key)
}

// GetInt8 collects the string value of a key if it exists.
func (c *context) GetInt8(key interface{}) (int8, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetInt8(key)
}

// GetInt16 collects the string value of a key if it exists.
func (c *context) GetInt16(key interface{}) (int16, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetInt16(key)
}

// GetInt64 collects the string value of a key if it exists.
func (c *context) GetInt64(key interface{}) (int64, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetInt64(key)
}

// GetInt32 collects the string value of a key if it exists.
func (c *context) GetInt32(key interface{}) (int32, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetInt32(key)
}

// GetInt collects the string value of a key if it exists.
func (c *context) GetInt(key interface{}) (int, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetInt(key)
}

// GetString collects the string value of a key if it exists.
func (c *context) GetString(key interface{}) (string, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.fields.GetString(key)
}
