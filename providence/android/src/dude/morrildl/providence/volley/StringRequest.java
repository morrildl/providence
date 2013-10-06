package dude.morrildl.providence.volley;

import java.util.HashMap;
import java.util.Map;

import com.android.volley.Response.ErrorListener;
import com.android.volley.Response.Listener;

public class StringRequest extends com.android.volley.toolbox.StringRequest {
    private String body = "";
    private String contentType = "text/plain";
    private HashMap<String, String> headers = new HashMap<String, String>();

    public StringRequest(int method, String url, Listener<String> listener,
            ErrorListener errorListener) {
        super(method, url, listener, errorListener);
    }

    @Override
    public byte[] getBody() {
        return body.getBytes();
    }

    @Override
    public String getBodyContentType() {
        return contentType;
    }

    @Override
    public Map<String, String> getHeaders() {
        return headers;
    }

    public void setBody(String body) {
        this.body = body;
    }

    public void setBodyContentType(String contentType) {
        this.contentType = contentType;
    }

    public void setHeader(String key, String value) {
        headers.put(key, value);
    }
}
