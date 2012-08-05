package dude.morrildl.providence.gcm;

import android.content.Context;

public class GCMBroadcastReceiver extends
		com.google.android.gcm.GCMBroadcastReceiver {
	@Override
	protected String getGCMIntentServiceClassName(Context context) {
		return "dude.morrildl.providence.gcm.GCMIntentService";
	}
}
