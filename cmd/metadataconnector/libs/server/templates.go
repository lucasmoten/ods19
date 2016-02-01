package server

var pageTemplateStart = `
<!DOCTYPE html>
<html lang='en'>
  <head>
    <title>Object-Drive</title>
    <meta charset='utf-8'>
    <base href="https://twl-server-generic2:8080/service/metadataconnector/1.0/" />
  </head>
	<body>
    <a href="">Return Home</a>
    <br />
		Method: %s
		<br />
		Distinguished Name:%s
		<br />
		<hr />
`

var pageTemplateEnd = `
	</body>
</html>
`
var pageTemplatePager = `
  <table id="%s" width="100%" />
`

var pageTemplateDataTable = `
  <table id="%s" width="100%" />
`
