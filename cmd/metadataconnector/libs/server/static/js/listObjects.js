
var __state = {};

var BASE_SERVICE_URL = '/service/metadataconnector/1.0/'


function getCN(dn) {
  return dn.substring(dn.indexOf('=')+1, dn.indexOf(','))
}

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
    url = BASE_SERVICE_URL + 'object/' + __state.parentId + '/list'
  }
  reqwest({
      url: url
    , method: 'post'
    , type: 'json'
    , contentType: 'application/json'
    , data: { pageNumber: 1, pageSize: 20, parentId: __state.parentId }
    , success: function (resp) {
      $.when(listUsers()).done(function (userdata) {

        __state.users = userdata[0];
        __state.Objects = resp.Objects;

        $.each(resp.Objects, function(index, item){
          $('#listObjectResults').append(_renderListObjectRow(index, item));
        })

        }).then(function(){
        // set up share handlers
          for ( var i = 0; i < resp.Objects.length; i++ ) {
            (function (_rowId, _obj) {
              $(_rowId).siblings('.shareButton').click(function() {
                var dn = $(_rowId).val();
                doShare(_obj.id, dn)
              })
            })('#listObjectRow_' + i, __state.Objects[i])
          }
          for ( var i = 0; i < resp.Objects.length; i++ ) {
            (function (_drowId, _obj) {
              $(_drowId).click(function() {
                doDelete(_obj.id, _obj.changeToken)
              })
            })('#deleteObjectRow_' + i, __state.Objects[i])
          }
      });
    }
  })
};

function refreshSharedWithMe() {
  $('#listSharedWithMeResults tbody > tr').remove();
  $.ajax({
    url: BASE_SERVICE_URL + 'shares',
    contentType: 'application/json',
    method: 'GET',
    success: function(resp) {
      _renderSharedWithMeTable(resp.Objects);
    }
  });
};


function _renderSharedWithMeTable(objs) {
  $.each(objs, function(index, item){
    $('#listSharedWithMeResults').append(_renderSharedWithMeRow(index, item));
  })
}

function _renderSharedWithMeRow(index, item) {

  var name = _renderObjectLink(item);
  var type = '<td>' + item.contentType + '</td>';
  var createdDate = '<td>' + item.createdDate + '</td>';
  var createdBy = '<td>' + getCN(item.createdBy) + '</td>';
  var size = '<td>' + item.contentSize + '</td>';
  var changeToken = '<td>' + item.changeToken + '</td>';
  var acm = '<td>' + item.acm + '</td>';

  return '<tr>' +
     name + type + createdDate + createdBy + size + changeToken + acm +
     '</tr>';
}

function doShare(objectId, userId, opts) {
  if (!opts) {
    opts = { create: true, read: true, update: true, delete: true}
  }
  var data = {
    grantee: userId,
    create: opts.create,
    read: opts.read,
    update: opts.update,
    delete: opts.delete
  };
  $.ajax({
    url: BASE_SERVICE_URL + 'object/' + objectId + '/share',
    contentType: 'application/json',
    method: 'POST',
    data: JSON.stringify(data),
    success: function(resp) {
      refreshSharedWithMe();
    }
  });
};

function doDelete(objectId, changeToken) {
  var folderName = $("#folderNameInput").val();
  var data = {
    changeToken: changeToken
  }
  $.ajax({
      url: '/service/metadataconnector/1.0/object/'+objectId,
      method: 'DELETE',
      data: JSON.stringify(data),
      contentType: 'application/json',
      success: function(data){
        refreshListObjects();
      }
  });
}


// Return a <tr> string suitable to append to table.
function _renderListObjectRow(index, item, elm) {

  var rowId = 'listObjectRow_' + index;
  var drowId = 'deleteObjectRow_' + index;

  var name = _renderObjectLink(item);
  var type = '<td>' + item.contentType + '</td>';
  var createdDate = '<td>' + item.createdDate + '</td>';
  var createdBy = '<td>' + getCN(item.createdBy) + '</td>';
  var size = '<td>' + item.contentSize + '</td>';
  var changeToken = '<td>' + item.changeToken + '</td>';
  var acm = '<td>' + item.acm + '</td>';
  var shareDropdown = _renderUsersDropdown(item, __state.users, rowId);
  var deleteButton = _renderDeleteButton(item, __state.users, drowId);

  return '<tr>' +
     name + type + createdDate + createdBy + size + changeToken + acm + shareDropdown + deleteButton +
     '</tr>';
}

function _renderUsersDropdown(obj, users, rowId) {
  var sel = $('<select></select>');
   sel.append($("<option>").attr('value', '').text('--'));
  for ( i=0; i < users.length; i ++ ) {
   sel.append($("<option>").attr('value', users[i].distinguishedName).text(getCN(users[i].distinguishedName)));
  }
  return '<td><select id="' + rowId + '">' + sel.html() + '</select><button class="shareButton">share</button></td>'
};

function _renderDeleteButton(obj, users, drowId) {
  return '<td><button id="'+drowId+'" class="deleteButton">delete</button></td>'
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

function shareTo() {};

function init() {
  // Set up click handlers
  $("#submitCreateObject").click(createObject);
  $("#refreshListObjects").click(refreshListObjects);
  $("#submitCreateFolder").click(createFolder);
  $("#refreshSharedWithMe").click(refreshSharedWithMe);

  // Get parentId from hidden field, if set.
  __state.parentId = $('#hiddenParentId').attr('data-value') || "";
  __state.users = [];

  refreshListObjects();
  refreshSharedWithMe();

};

$(document).ready(init);
