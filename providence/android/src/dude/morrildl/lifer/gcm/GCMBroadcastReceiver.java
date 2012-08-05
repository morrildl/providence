package dude.morrildl.lifer.gcm;

import android.content.Context;

public class GCMBroadcastReceiver extends
		com.google.android.gcm.GCMBroadcastReceiver {
	@Override
	protected String getGCMIntentServiceClassName(Context context) {
		return "dude.morrildl.lifer.gcm.GCMIntentService";
	}
}
