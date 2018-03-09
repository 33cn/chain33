addr = "http://192.168.0.138:8081/getInformation"

ty = "raw"
resp ,err = http.Post(addr, ty, strings.NewReader("Corp1"))

body, err = ioutil.ReadAll(resp.Body)
newbody = body[2:]
println(newbody)

m = make(map[string]string)
m,err = json.Unmarshal(newbody)
println(m)