
// TODO change this after we merge in Rob's request object code
function doRequest() {

  var t = $('#listObjectResults');

  reqwest({
      url: '/service/metadataconnector/1.0/objects'
    , method: 'post'
    , type: 'json'
    , contentType: 'application/json'
    , data: { pageNumber: '1', pageSize: 20 }
    , success: function (resp) {
        // qwery('#listObjectsResults tr:last').html(resp)

        $.each(resp.Objects, function(index, item){
        console.log(item.Name)
        // Name	Type	Created Date	Created By	Size	ACM
        var name = '<td><a href=' + item.URL + '/stream>' + item.Name + '</a></td>';
        var type = '<td>' + item.Type + '</td>';
        var createdDate = '<td>' + item.CreatedDate + '</td>';
        var createdBy = '<td>' + item.CreatedBy + '</td>';
        var size = '<td>' + item.Size + '</td>';
        var changeToken = '<td>' + item.ChangeToken + '</td>';
        var acm = '<td>' + item.ACM + '</td>';
        console.log('<tr>' + name + type + createdDate + createdBy + size + changeToken + acm + '</tr>');
         $('#listObjectResults').append('<tr>' + name + type + createdDate + createdBy + size + changeToken + acm + '</tr>');
        // t.append('<tr>' + name + type + createdDate + createdBy + size + acm + '</tr>');
        })

      }
  })
};

$("#submitCreateObject").click(function() {
  // get the form data
  var objectName = $("#newObjectName").val();
  var classification = $("#classification").val();
  var jsFileObject = $("#fileHandle")[0].files[0];
  var mimeType = jsFileObject.type || "text/plain";
  var fileName = jsFileObject.name;
  var size = jsFileObject.size;

  var req = {
    classification: classification,
    title: objectName,
    fileName: fileName,
    size: size,
    mimeType: mimeType
  }

  // call the server with the data
  var formData = new FormData();
  formData.append("CreateObjectRequest", JSON.stringify(req));
  formData.append("filestream", jsFileObject);

});

function submitCreateObject(data) {
  console.log("submitCreateObject called!")
};

document.onload = doRequest();


/*

*/
