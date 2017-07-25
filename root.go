package main

import "html/template"

var tmplRoot = template.Must(template.New("root").Parse(`
<html>
<script>
var chunk_size = 100000;

function status(s) {
  console.log(s);
  var o = document.getElementById("status");
  o.innerText = s;
}

function upload_part(f, ws, offset)
{
  var reader = new FileReader();
  var s = f.slice(offset, offset + chunk_size);
  reader.onload = function() {
    if (ws.readyState === ws.CLOSED) {
      return;
    }
    var data = reader.result;
    var size = data.byteLength;
    //console.log(data);
    //console.log("loaded part offset " + offset + " size: " + size);
    if (size > 0) {
      var sent = offset + size;
      // status("Sent " + sent + " of " + f.size + " " + (100.0*sent/f.size) + "%");
      ws.send(data);
      upload_part(f, ws, offset + size);
    } else {
      // status("All in flight...");
    }
  }
  reader.readAsArrayBuffer(s);
}

function upload(event)
{
  var fs = document.getElementById("files");
  var f = fs.files[0];
  var error_set = false;
  console.log(f);
  ws = new WebSocket("wss://"+window.location.host+"{{.}}upload-ws/" + f.name);
  ws.onerror = function(err) {
    console.log("Websocket error: " + err);
  }
  ws.onopen = function() {
    upload_part(f, ws, 0);
  }
  ws.onerror = function(e) {
    console.log("Error: " + e);
  }
  ws.onclose = function(ev) {
    if (ev.code != 1000) {
      if (!error_set) {
        status("Upload failed.");
      }
      console.log(ev);
    } else {
      status("Upload done");
      document.getElementById("progress").value = 100;
    }
  }
  ws.onmessage = function(ev) {
    var o = JSON.parse(ev.data);
    if (o.Error !== undefined) {
      status("Upload failed: " + o.Error);
      error_set = true;
    }
    if (o.Written !== undefined) {
      var p = 100.0*o.Written/f.size;
      status("Sent " + o.Written + " of " + f.size + " " + Math.round(p) + "%");
      document.getElementById("progress").value = p;
      if (o.Written == f.size) {
        ws.close(1000)
      }
    }
  }
}
</script>
<table style="width: 100%">
  <tr>
<!--    <th>File</th> -->
    <th>Status</th>
    <th>Progress</th>
  </tr>
  <tr>
<!--    <td>TODO</td> -->
    <td style="width: 30%; text-align: right;" id="status">IDLE</td>
    <td><progress style="width: 100%" id="progress" value="0" max="100" /></td>
  </tr>
</table>
<input id="files" type="file" name="file" accept="*/*" />
<input type="submit" onclick="upload()" />
</form>
</html>
`))
