package dude.morrildl.lifer;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import android.widget.TextView;

import com.google.android.gcm.GCMRegistrar;

public class LiferActivity extends Activity {
	/** Called when the activity is first created. */
	@Override
	public void onCreate(Bundle savedInstanceState) {
		super.onCreate(savedInstanceState);
		setContentView(R.layout.main);

		GCMRegistrar.checkDevice(this);
		GCMRegistrar.checkManifest(this);
		String regId = GCMRegistrar.getRegistrationId(this);
		if (regId == null || "".equals(regId)) {
			GCMRegistrar.register(this, GCMIntentService.SENDER_ID);
		}
	}

	@Override
	public void onResume() {
		super.onResume();
		String regId = GCMRegistrar.getRegistrationId(this);
		if (regId != null && !"".equals(regId)) {
			((TextView) findViewById(R.id.reg_id)).setText(regId);
			Log.e("booga", regId);
		} else {
			((TextView) findViewById(R.id.reg_id)).setText("no reg ID");			
		}
		
		Bundle b = getIntent().getExtras();
		if (b != null) {
			StringBuilder sb = new StringBuilder();
			for (String k : b.keySet()) {
				sb.append(k).append(" = ").append(b.getString(k)).append("\n");
			}
			((TextView)findViewById(R.id.main_text)).setText(sb.toString());
		}
	}
}