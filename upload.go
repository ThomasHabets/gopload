package main

import "html/template"

var tmplUpload = template.Must(template.New("root").Parse(`
<html>
Upload complete!
<a href="{{.}}">Back</a>
</html>
`))
