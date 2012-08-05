package dude.morrildl.lifer.photos;

import android.net.Uri;
import android.util.Log;

public class PhotoUploaderJobber extends Jobber {
	private Uri uri;
	@SuppressWarnings("unused") // we only work if given a URI
	private PhotoUploaderJobber() {	}
	public PhotoUploaderJobber(Uri uri) {
		this.uri = uri;
	}
	@Override
	public void run() {
		Log.e("PhotoUploaderJobber", "run with " + uri.toString());
	}
}
