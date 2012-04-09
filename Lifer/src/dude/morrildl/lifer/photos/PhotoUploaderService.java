package dude.morrildl.lifer.photos;

import android.app.Service;
import android.content.Intent;
import android.net.Uri;
import android.os.Binder;
import android.os.IBinder;
import android.util.Log;

public class PhotoUploaderService extends Service {
	public static final String OPERATION = "dude.morrildl.lifer.photoupload.operation";
	public static final int OPERATION_NOOP = 0;
	public static final int OPERATION_UPLOAD_PHOTO_URI = 1;

	private final LocalBinder localBinder = new LocalBinder();

	public class LocalBinder extends Binder {
		public PhotoUploaderService getService() {
			return PhotoUploaderService.this;
		}
	}

	@Override
	public IBinder onBind(Intent intent) {
		return localBinder;
	}

	public void uploadPhotoUri(Uri uri) {

	}

	@Override
	public void onCreate() {
		super.onCreate();
	}

	@Override
	public int onStartCommand(Intent intent, int flags, int startId) {
		super.onStartCommand(intent, flags, startId);
		if (intent.getExtras().getInt(OPERATION) == OPERATION_UPLOAD_PHOTO_URI) {
			Uri uri = intent.getData();
			Log.e("PhotoUploaderService.onStartCommand", uri == null ? "empty"
					: uri.toString());
		} else {
			Log.e("PhotoUploaderService.onStartCommand", "unknown operation " + intent.getExtras().getInt(OPERATION));
		}
		return Service.START_NOT_STICKY;
	}
}
