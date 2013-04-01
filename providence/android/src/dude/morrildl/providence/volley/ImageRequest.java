package dude.morrildl.providence.volley;

import java.util.HashMap;
import java.util.Map;

import android.graphics.Bitmap;
import android.graphics.Bitmap.Config;

import com.android.volley.Response.ErrorListener;
import com.android.volley.Response.Listener;

public class ImageRequest extends com.android.volley.toolbox.ImageRequest {

    private HashMap<String, String> headers;

    public ImageRequest(String url, Listener<Bitmap> listener, int maxWidth, int maxHeight,
            Config decodeConfig, ErrorListener errorListener) {
        super(url, listener, maxWidth, maxHeight, decodeConfig, errorListener);
        headers = new HashMap<String, String>();
    }
    
    public Map<String, String> getHeaders() {
        return headers;
    }
    
    public void setHeader(String key, String value) {
        headers.put(key, value);
    }
}
