

var __state = {};


function refreshListObjects() {
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
        var name = '<td><a href=' + item.url + '/stream>' + item.name + '</a></td>';
        var type = '<td>' + item.contentType + '</td>';
        var createdDate = '<td>' + item.createdDate + '</td>';
        var createdBy = '<td>' + item.createdBy + '</td>';
        var size = '<td>' + item.contentSize + '</td>';
        var changeToken = '<td>' + item.changeToken + '</td>';
        var acm = '<td>' + item.acm + '</td>';
        console.log('<tr>' + name + type + createdDate + createdBy + size + changeToken + acm + '</tr>');
         $('#listObjectResults').append('<tr>' + name + type + createdDate + createdBy + size + changeToken + acm + '</tr>');
        // t.append('<tr>' + name + type + createdDate + createdBy + size + acm + '</tr>');
        })

      }
  })
};

function createObject() {
      console.log("createObject called");
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

      $.ajax({
      url: '/service/metadataconnector/1.0/object',
      data: formData,
      cache: false,
      contentType: false,
      processData: false,
      type: 'POST',
      success: function(data){
        console.log("We did it!")
        console.log(data);
      }
  });
}

function init() {
  // Set up click handlers
  $("#submitCreateObject").click(createObject);
  $("#refreshListObjects").click(refreshListObjects)

  // initial state
  __state.parentId = "";
};

$(document).ready(init);
