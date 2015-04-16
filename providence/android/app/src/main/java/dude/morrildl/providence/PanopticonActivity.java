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
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.URL;

import javax.net.ssl.HttpsURLConnection;

import android.app.Activity;
import android.app.AlertDialog;
import android.app.FragmentManager;
import android.app.FragmentTransaction;
import android.content.ActivityNotFoundException;
import android.content.Context;
import android.content.DialogInterface;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.content.pm.PackageManager.NameNotFoundException;
import android.content.res.Resources.NotFoundException;
import android.database.sqlite.SQLiteDatabase;
import android.net.Uri;
import android.os.AsyncTask;
import android.os.Bundle;
import android.util.Log;
import android.view.Menu;
import android.view.MenuInflater;
import android.view.MenuItem;

import com.google.android.gcm.GCMRegistrar;

import dude.morrildl.providence.db.OpenHelper;
import dude.morrildl.providence.gcm.GCMIntentService;

public class PanopticonActivity extends Activity {
    private static final String ZXING_PACKAGE = "com.google.zxing.client.android";

    private Config config;

    private EventHistoryFragment ehf;

    private ConfigWaitFragment cwf;

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (data != null) {
            Config.storeConfig(this, data.getStringExtra("SCAN_RESULT"));
        } else {
            Log.e("PanopticonActivity.onActivityResult",
                    "no data returned by Barcode Scanner?");
            return;
        }

        if (config.isReady()) {
            // isReady() will attempt to reload config data if necessary
            FragmentManager fm = getFragmentManager();
            FragmentTransaction ft = fm.beginTransaction();
            EventHistoryFragment ehf = new EventHistoryFragment();
            ft.add(R.id.main_container, ehf).show(ehf).commit();
        } else {
            Log.e("PanopticonActivity.onActivityResult",
                    "config not ready even after QR scan");
        }
    }

    /** Called when the activity is first created. */
    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        GCMRegistrar.checkDevice(this);
        GCMRegistrar.checkManifest(this);
        String regId = GCMRegistrar.getRegistrationId(this);
        if (regId == null || "".equals(regId)) {
            GCMRegistrar.register(this, GCMIntentService.SENDER_ID);
        }

        config = Config.getInstance(this);

        setContentView(R.layout.main);
        ehf = new EventHistoryFragment();
        cwf = new ConfigWaitFragment();
        getFragmentManager().beginTransaction().add(R.id.main_container, ehf)
                .add(R.id.main_container, cwf).commit();
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        MenuInflater inflater = getMenuInflater();
        inflater.inflate(R.menu.providence, menu);
        return true;
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        GCMRegistrar.onDestroy(this);
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        // Handle item selection
        switch (item.getItemId()) {
        case R.id.menu_reset:
            AlertDialog.Builder ad = new AlertDialog.Builder(this);
            ad.setTitle(R.string.mn_cd_confirm_title);
            ad.setMessage(R.string.mn_cd_confirm_body);
            ad.setPositiveButton(R.string.mn_cd_confirm_pos,
                    new DialogInterface.OnClickListener() {
                        @Override
                        public void onClick(DialogInterface di, int i) {
                            Context context = PanopticonActivity.this;
                            Config.clearConfig(context);
                            OpenHelper helper = new OpenHelper(context);
                            SQLiteDatabase db = helper.getWritableDatabase();
                            db.delete("events", null, null);
                            finish();
                        }
                    });
            ad.setNegativeButton(R.string.mn_cd_confirm_neg,
                    new DialogInterface.OnClickListener() {
                        @Override
                        public void onClick(DialogInterface di, int i) {
                        }
                    });
            ad.show();

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
                (new AsyncTask<String, Integer, String>() {
                    @Override
                    protected String doInBackground(String... regIds) {
                        String regId = regIds[0];
                        URL url;
                        HttpURLConnection cxn = null;
                        try {
                            Context context = PanopticonActivity.this;
                            Config config = Config.getInstance(context);
                            if (!config.isReady()) {
                                return "";
                            }
                            url = new URL(config.getRegIdUrl());
                            cxn = (HttpURLConnection) url.openConnection();

                            NetworkHelper helper = NetworkHelper
                                    .getInstance(context);
                            String token = helper.fetchAuthToken();
                            cxn.addRequestProperty("X-OAuth-JWT", token);

                            if (cxn instanceof HttpsURLConnection) {
                                ((HttpsURLConnection) cxn)
                                        .setSSLSocketFactory(helper
                                                .getSslSocketFactory());
                            }

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
                        } catch (OAuthException e) {
                            Log.e("doInBackground",
                                    "failed fetching auth token", e);
                        } finally {
                            if (cxn != null)
                                cxn.disconnect();
                        }
                        return "";
                    }

                    @Override
                    protected void onPostExecute(String result) {
                        if (result != null && "OK".equals(result.trim())) {
                            GCMRegistrar.setRegisteredOnServer(
                                    PanopticonActivity.this, true);
                        } else {
                            Log.w("onPostExecute result",
                                    "failed registering GCM RegID on server");
                        }
                    }
                }).execute(regId);
            }
        }

        FragmentManager fm = getFragmentManager();

        if (!config.isReady()) {
            // i.e. if we have no local config

            // first show a static placeholder "please wait" screen
            fm.beginTransaction().hide(ehf).show(cwf).commit();

            // if Barcode scanner isn't installed, pop a dialog to install it
            PackageManager pm = getPackageManager();
            try {
                pm.getApplicationInfo(ZXING_PACKAGE, 0);
            } catch (NameNotFoundException e) {
                AlertDialog.Builder ad = new AlertDialog.Builder(this);
                ad.setTitle(R.string.bc_dialog_title);
                ad.setMessage(R.string.bc_dialog_body);
                ad.setPositiveButton(R.string.bc_confirm,
                        new DialogInterface.OnClickListener() {
                            @Override
                            public void onClick(DialogInterface di, int i) {
                                String packageName = ZXING_PACKAGE;
                                Uri uri = Uri
                                        .parse("https://play.google.com/store/apps/details?id="
                                                + packageName);
                                Intent intent = new Intent(Intent.ACTION_VIEW,
                                        uri);
                                try {
                                    startActivity(intent);
                                } catch (ActivityNotFoundException e) {
                                    Log.e("PanopticonActivity.onCreate",
                                            "Google Play Store not present?!");
                                    finish();
                                }
                            }
                        });
                ad.setNegativeButton(R.string.bc_reject,
                        new DialogInterface.OnClickListener() {
                            @Override
                            public void onClick(
                                    DialogInterface dialogInterface, int i) {
                                finish();
                            }
                        });
                ad.show();

                return;
            }

            // Barcode Scanner is installed; now launch it to scan a QR
            Intent i = new Intent(ZXING_PACKAGE + ".SCAN");
            i.addCategory(Intent.CATEGORY_DEFAULT);
            i.putExtra("SCAN_FORMATS", "QR_CODE");
            i.setPackage(ZXING_PACKAGE);
            i.addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP);
            i.addFlags(Intent.FLAG_ACTIVITY_CLEAR_WHEN_TASK_RESET);
            startActivityForResult(i, 42);

            return;
        }

        // i.e. Config is ready on first try -- copasetic case
        fm.beginTransaction().hide(cwf).show(ehf).commit();
    }
}