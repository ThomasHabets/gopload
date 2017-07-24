package main

import "html/template"

var tmplRoot = template.Must(template.New("root").Parse(`
<html>
<script>
var chunk_size = 1000000;

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
    var data = reader.result;
    var size = data.byteLength;
    //console.log(data);
    //console.log("loaded part offset " + offset + " size: " + size);
    if (size > 0) {
      var sent = offset + size;
      status("Sent " + sent + " of " + f.size + " " + (100.0*sent/f.size) + "%");
      ws.send(data);
      upload_part(f, ws, offset + size);
    } else {
      status("All in flight...");
      ws.close(1000)
    }
  }
  reader.readAsArrayBuffer(s);
}

function upload(event)
{
  var fs = document.getElementById("files");
  var f = fs.files[0];
  console.log(f);
  ws = new WebSocket("wss://"+window.location.host+"{{.}}upload-ws/" + f.name);
  ws.onerror = function(err) {
    console.log("Websocket error: " + err);
  }
  ws.onopen = function() {
    upload_part(f, ws, 0);
  }
  ws.onclose = function(ev) {
    if (ev.code != 1000) {
      status("Upload failed: " + ev);
    } else {
      status("Upload done");
    }
  }
}
</script>
<table>
  <tr>
    <th>File</th>
    <th>Status</th>
  </tr>
  <tr>
    <td>TODO</td>
    <td id="status">IDLE</td>
  </tr>
</table>
<input id="files" type="file" name="file" accept="*/*" />
<input type="submit" onclick="upload()" />
</form>
</html>
`))
