
var __state = {};

var BASE_SERVICE_URL = '/service/metadataconnector/1.0/'

function newParent(id) {
  __state.parentId = id;
  refreshListObjects();
};

function listUsers() {
  return $.ajax({
    url: '/service/metadataconnector/1.0/users',
    contentType: 'application/json',
    method: 'GET'
  });
};

function refreshListObjects() {
  var url;
  var t = $('#listObjectResults');
  // remove children first
  $('#listObjectResults tbody > tr').remove();
  // choose correct listObjects URL
  if (__state.parentId === "") {
    url  = BASE_SERVICE_URL + 'objects';
  } else {
    url = '/service/metadataconnector/1.0/object/' + __state.parentId + '/list'
  }
  console.log('Requesting...' + url);
  reqwest({
      url: url
    , method: 'post'
    , type: 'json'
    , contentType: 'application/json'
    , data: { pageNumber: '1', pageSize: 20, parentId: __state.parentId }
    , success: function (resp) {
      $.when(listUsers()).done(function (udata) {

        console.log('got users');
        console.log(udata[0]);

        $.each(resp.Objects, function(index, item){
          // render each row
          $('#listObjectResults').append(_renderListObjectRow(index, item));
        })

      }

      );
    }
  })
};

function refreshSharedWithMe() {
      // todo
      // $.ajax({
      //   url: '/service/metadataconnector/1.0/object',
      //   data: formData,
      //   contentType: 'application/json',
      //   method: 'POST',
      //   success: function(data){
      //     console.log("Got shares.")
      //     // render share data
      //   }
      // });

};

// Return a <tr> string suitable to append to table.
function _renderListObjectRow(index, item) {

  // make a rowId to identify the row
  var rowId = 'listObjectRow_' + index;

  // Name	Type	Created Date	Created By	Size	ACM
  var name = _renderObjectLink(item);
  var type = '<td>' + item.contentType + '</td>';
  var createdDate = '<td>' + item.createdDate + '</td>';
  var createdBy = '<td>' + item.createdBy + '</td>';
  var size = '<td>' + item.contentSize + '</td>';
  var changeToken = '<td>' + item.changeToken + '</td>';
  var acm = '<td>' + item.acm + '</td>';
  var shareTo = '<td>' + item.acm + '</td>';
  return '<tr id="' + rowId + '" >' + 
     name + type + createdDate + createdBy + size + changeToken + acm
     + '</tr>'
}



// Render a proper href, depending on whether an object is a folder or an object proper.
function _renderObjectLink(item) {
  var link;
  if (item.typeName === "Folder") {
    link = '<td><a href="'+ BASE_SERVICE_URL + 'home/listObjects?parentId=' + item.id + '">'+item.name+'</a></td>';
  } else {
    link = '<td><a href="'+ BASE_SERVICE_URL + 'object/' + item.id + '/stream">' + item.name + '</a></td>';
  }
  return link;
}

function createObject() {
      console.log("createObject called");
      // get the form data
      var objectName = $("#newObjectName").val();
      var classification = $("#classification").val();
      var jsFileObject = $("#fileHandle")[0].files[0];
      var contentType = jsFileObject.type || "text/plain";
      var fileName = jsFileObject.name;
      var size = jsFileObject.size;
      var rawAcm = '{"version":"2.1.0","classif":"'+classification+'"}'
      var req = {
        acm: rawAcm,
        title: objectName,
        fileName: fileName,
        size: size,
        contentType: contentType,
        parentId: __state.parentId
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
        method: 'POST',
        success: function(data){
          console.log("We did it!")
          console.log(data);
          refreshListObjects();
        }
      });
}

function createFolder() {
  var folderName = $("#folderNameInput").val();
  var data = {
    typeName: "Folder",
    parentId: __state.parentId,
    name: folderName
  }
      $.ajax({
      url: '/service/metadataconnector/1.0/folder',
      data: JSON.stringify(data),
      method: 'POST',
      contentType: 'application/json',
      success: function(data){
        console.log("createFolder success.")
        console.log(data);
        refreshListObjects();
      }
  });
}

function init() {
  // Set up click handlers
  $("#submitCreateObject").click(createObject);
  $("#refreshListObjects").click(refreshListObjects);
  $("#submitCreateFolder").click(createFolder);

  // Get parentId from hidden field, if set.
  __state.parentId = $('#hiddenParentId').attr('data-value') || "";
  __state.users = [];

  refreshListObjects();
  refreshSharedWithMe();

};

$(document).ready(init);
