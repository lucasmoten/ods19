package libs

/*
 * These are the templates to give a basic user interface.
 */

//IndexForm leads to the upload form
var IndexForm = `
<html>
	<head><title>OD Uploader</title>
	<body>
		<a href='/upload'>Upload</a>
	</body>
</html>
`

//UploadForm is the user interface element to post data without programming
var UploadForm = `
<html>
  <head><title>Raw Uploader</title>
	<body>
		UploadBy:%s
		<br>
		%s
		<br>
		<form action='/upload' method='POST' enctype='multipart/form-data'>
			<select name='classification'>
				<option value='U'>Unclassified</option>
				<option value='C'>Classified</option>
				<option value='S'>Secret</option>
				<option value='T'>Top Secret</option>
			</select>
			The File:<input name='theFile' type='file'>
			<input type='submit'>
		</form>
	</body>
</html>
`
