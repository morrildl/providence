package dude.morrildl.providence.photos;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

public class NewPhotoReceiver extends BroadcastReceiver {
	@Override
	public void onReceive(Context context, Intent intent) {
		Intent serviceIntent = new Intent(context, PhotoUploaderService.class);
		serviceIntent.setData(intent.getData());
		serviceIntent.putExtra(PhotoUploaderService.OPERATION, PhotoUploaderService.OPERATION_UPLOAD_PHOTO_URI);
		context.startService(serviceIntent);
	}
}
