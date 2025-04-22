package utilbase

import "encoding/json"
func ToJson(v interface{})string{
    r ,_ := json.Marshal(v)
    return string(r)
}