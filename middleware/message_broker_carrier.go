package middleware

type MessageBrokerCarrier map[string]interface{}

func (c MessageBrokerCarrier) Get(k string) string {
	if v, ok := c[k]; ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return ""
}
func (c MessageBrokerCarrier) Set(k, v string) { c[k] = v }
func (c MessageBrokerCarrier) Keys() []string {
	ks := make([]string, 0, len(c))
	for k := range c {
		ks = append(ks, k)
	}
	return ks
}
