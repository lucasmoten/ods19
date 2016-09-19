package server

var pageTemplateStart = `
<!DOCTYPE html>
<html lang='en'>
  <head>
    <title>Object-Drive</title>
    <meta charset='utf-8'>
    <base href="%s" />
  </head>
	<body>
    <a href="">Return Home</a>
    <br />
		<h2> %s </h2>
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