/* Copyright © 2012 Dan Morrill
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package dude.morrildl.providence;

import java.io.IOException;
import java.io.OutputStream;
import java.net.MalformedURLException;
import java.net.URL;
import java.security.KeyStoreException;

import javax.net.ssl.HttpsURLConnection;

import android.app.Activity;
import android.app.FragmentManager;
import android.app.FragmentTransaction;
import android.content.Context;
import android.content.res.Resources.NotFoundException;
import android.os.AsyncTask;
import android.os.Bundle;
import android.util.Log;
import android.view.Menu;
import android.view.MenuInflater;
import android.view.MenuItem;
import android.widget.Toast;

import com.google.android.gcm.GCMRegistrar;

import dude.morrildl.providence.gcm.GCMIntentService;
import dude.morrildl.providence.panopticon.EventDetailsFragment;
import dude.morrildl.providence.panopticon.EventHistoryFragment;
import dude.morrildl.providence.util.Network;

public class PanopticonActivity extends Activity /*
												 * implements
												 * ActionBar.OnNavigationListener
												 */{
	private EventDetailsFragment eventDetailsFragment = null;
	private EventHistoryFragment eventHistoryFragment = null;
	private Network networkUtil;
	//private static final String GAUTH_CLIENT_ID = "25235963451.apps.googleusercontent.com";

	/** Called when the activity is first created. */
	@Override
	public void onCreate(Bundle savedInstanceState) {
		super.onCreate(savedInstanceState);
		setContentView(R.layout.main);
		FragmentManager fm = getFragmentManager();
		FragmentTransaction ft = fm.beginTransaction();
		eventDetailsFragment = new EventDetailsFragment();
		eventHistoryFragment = new EventHistoryFragment();
		ft.add(R.id.main_container, eventHistoryFragment);
		ft.add(R.id.main_container, eventDetailsFragment);
		ft.show(eventHistoryFragment).hide(eventDetailsFragment);
		ft.commit();

		GCMRegistrar.checkDevice(this);
		GCMRegistrar.checkManifest(this);
		String regId = GCMRegistrar.getRegistrationId(this);
		if (regId == null || "".equals(regId)) {
			GCMRegistrar.register(this, GCMIntentService.SENDER_ID);
		}
		
		try {
			networkUtil = Network.getInstance(this);
		} catch (KeyStoreException e) {
			Log.e("PanopticonActivity.onCreate", "FATAL: exception loading the network utility");
			throw new RuntimeException("exception loading the network utility", e);
		}
	}

	@Override
	protected void onDestroy() {
		super.onDestroy();
		GCMRegistrar.onDestroy(this);
	}

	@Override
	public boolean onCreateOptionsMenu(Menu menu) {
		MenuInflater inflater = getMenuInflater();
		inflater.inflate(R.menu.providence, menu);
		return true;
	}

	@Override
	public boolean onOptionsItemSelected(MenuItem item) {
		// Handle item selection
		switch (item.getItemId()) {
		case R.id.menu_test_1:
			Toast.makeText(this, R.string.menu_test_1, Toast.LENGTH_SHORT)
					.show();
			return true;
		case R.id.menu_test_2:
			Toast.makeText(this, R.string.menu_test_2, Toast.LENGTH_SHORT)
					.show();
			return true;
		default:
			return super.onOptionsItemSelected(item);
		}
	}

	@Override
	public void onResume() {
		super.onResume();
		String regId = GCMRegistrar.getRegistrationId(this);
		if (regId != null && !"".equals(regId)) {
			if (!GCMRegistrar.isRegisteredOnServer(this)) {
				new ServerRegisterTask(this).execute(regId);
			}
		}
	}

	class ServerRegisterTask extends AsyncTask<String, Integer, String> {
		private Context context;

		@SuppressWarnings("unused")
		private ServerRegisterTask() {
		}

		public ServerRegisterTask(Context context) {
			this.context = context;
		}

		@Override
		protected String doInBackground(String... regIds) {
			String regId = regIds[0];
			URL url;
			HttpsURLConnection cxn = null;
			try {
				url = networkUtil.urlForResource(R.raw.regid_url, null);
				cxn = (HttpsURLConnection) url.openConnection();
				cxn.setSSLSocketFactory(networkUtil.getSslSocketFactory());
				cxn.setDoInput(true);
				cxn.setRequestMethod("POST");
				OutputStream os = cxn.getOutputStream();
				os.write(regId.getBytes());
				os.close();
				return new java.util.Scanner(cxn.getInputStream())
						.useDelimiter("\\A").next();
			} catch (MalformedURLException e) {
				Log.e("doInBackground", "URL error", e);
			} catch (NotFoundException e) {
				Log.e("doInBackground", "URL error", e);
			} catch (IOException e) {
				Log.e("doInBackground", "transmission error", e);
			} finally {
				if (cxn != null)
					cxn.disconnect();
			}
			return "";
		}

		@Override
		protected void onPostExecute(String result) {
			Log.e("onPostExecute result", "'" + result + "'");
			if (result != null && "OK".equals(result.trim()))
				GCMRegistrar.setRegisteredOnServer(context, true);
		}
	}

	public void showDetailsFragment(long id) {
		eventDetailsFragment.setId(id);
		FragmentManager fm = getFragmentManager();
		fm.beginTransaction().hide(eventHistoryFragment)
				.show(eventDetailsFragment).addToBackStack(null).commit();
	}
}