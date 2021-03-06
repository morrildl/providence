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

import java.text.DateFormat;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Calendar;
import java.util.Date;
import java.util.Locale;
import java.util.TimeZone;

import android.app.ListFragment;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ListView;
import android.widget.TextView;
import dude.morrildl.providence.db.OpenHelper;
import dude.morrildl.providence.view.PanopticonCursorAdapter;

public class EventHistoryFragment extends ListFragment {
    private Cursor c;
    private SQLiteDatabase db;

    private String getConditionalDateFrom(String timestamp)
            throws ParseException {
        SimpleDateFormat sdf = new SimpleDateFormat(
                "yyyy'-'MM'-'dd'T'HH:mm:ss", Locale.US);
        Date parsedTs = sdf.parse(timestamp);

        Calendar today = Calendar.getInstance(TimeZone.getDefault());
        Calendar midnight = Calendar.getInstance(TimeZone.getDefault());
        midnight.clear();
        midnight.set(today.get(Calendar.YEAR), today.get(Calendar.MONTH),
                today.get(Calendar.DATE));

        if (parsedTs.after(midnight.getTime())) {
            return "";
        }

        DateFormat df = SimpleDateFormat
                .getDateInstance(SimpleDateFormat.MEDIUM);
        return df.format(parsedTs);
    }

    private String getTimeFrom(String timestamp) throws ParseException {
        SimpleDateFormat sdf = new SimpleDateFormat(
                "yyyy'-'MM'-'dd'T'HH:mm:ss", Locale.US);
        Date parsedTs = sdf.parse(timestamp);

        DateFormat df = SimpleDateFormat
                .getTimeInstance(SimpleDateFormat.MEDIUM);
        return df.format(parsedTs);
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container,
            Bundle savedInstanceState) {
        return inflater.inflate(R.layout.event_history_fragment, container,
                false);
    }

    @Override
    public void onHiddenChanged(boolean hidden) {
        super.onHiddenChanged(hidden);
        if (!hidden) {
            OpenHelper helper = new OpenHelper(getActivity());
            db = helper.getReadableDatabase();
            c = db.query("events", new String[] { "which", "type", "event",
                    "ts", "rowid as _id", "eventid" },
                    "type <> 'Motion Sensor'", null, null, null, "ts desc");

            setListAdapter(new PanopticonCursorAdapter(getActivity(), c, 0));

            Cursor tmpCursor = db.query("last_motion",
                    new String[] { "max(ts)" }, null, null, null, null, null);
            TextView tv = (TextView) getView().findViewById(
                    R.id.motion_status_text);
            if (tv != null) {
                if (tmpCursor.moveToFirst()) {
                    try {
                        String timestamp = tmpCursor.getString(0);
                        if (timestamp != null) {
                            String timeString = getTimeFrom(timestamp);
                            String dateString = getConditionalDateFrom(timestamp);

                            if (!"".equals(dateString)) {
                                timeString = timeString + " on " + dateString;
                            }

                            tv.setText("Last motion detected at " + timeString);
                            tv.setVisibility(View.VISIBLE);
                        } else {
                            tv.setVisibility(View.GONE);
                        }
                    } catch (ParseException e) {
                        tv.setVisibility(View.GONE);
                    }
                } else {
                    tv.setVisibility(View.GONE);
                }
            }
        }
    }

    @Override
    public void onListItemClick(ListView l, View v, int position, long id) {
        super.onListItemClick(l, v, position, id);
    }

    public void onPause() {
        super.onPause();
        try {
            c.close();
            db.close();
        } catch (Throwable t) {
        }
    }

    public void onResume() {
        super.onResume();
        onHiddenChanged(false);
    }
}
