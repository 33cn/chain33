addr = "http://192.168.0.138:8081/putInformation"
ty = "raw"
info = "{	\"recordId\" :\"Corp1\",	\"negativeType\" : \"Overdue\",	\"negativeSeverity\" : \"Serious\",	\"negativeInfo\" : \"Vvvv\"}"
foo = {"recordId":"Corp1", "negativeType":"Overdue","negativeSeverity":"Serious","negativeInfo":"shasha"}
m,_ = json.Marshal(foo)
http.Post(addr, ty, bytes.NewReader(m))