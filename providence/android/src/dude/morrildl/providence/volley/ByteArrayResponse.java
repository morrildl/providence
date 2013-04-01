package dude.morrildl.providence.volley;

import java.util.Map;

public class ByteArrayResponse {
    public byte[] bytes;

    public Map<String, String> headers;

    public ByteArrayResponse() {
    }
    public ByteArrayResponse(byte[] bytes, Map<String, String> headers) {
        this.bytes = bytes;
        this.headers = headers;
    }
}
