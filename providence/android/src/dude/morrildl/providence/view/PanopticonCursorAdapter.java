/* Copyright © 2013 Dan Morrill
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
package dude.morrildl.providence.view;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.HashMap;
import java.util.Locale;

import android.content.Context;
import android.content.SharedPreferences;
import android.database.Cursor;
import android.graphics.Color;
import android.util.Log;
import android.view.View;
import android.view.ViewGroup;
import android.widget.CursorAdapter;
import dude.morrildl.providence.Stuff;

/**
 * ListView adapter that feeds data from a Sqlite cursor into EventSummaryViews.
 */
public class PanopticonCursorAdapter extends CursorAdapter {
    private final HashMap<String, Integer> colorMap = new HashMap<String, Integer>();
    private final int[] colors = new int[5];
    private final int defaultColor = Color.argb(0xff, 0x33, 0xb5, 0xe5);
    private final Stuff stuff;
    private int usedColors = 0;

    public PanopticonCursorAdapter(Context context, Stuff stuff, Cursor c,
            int flags) {
        super(context, c, flags);
        SharedPreferences prefs = context.getSharedPreferences("colormap",
                Context.MODE_PRIVATE);
        for (String key : prefs.getAll().keySet()) {
            colorMap.put(key, prefs.getInt(key, 0));
        }
        usedColors = colorMap.size();
        colors[0] = Color.argb(0xff, 0xff, 0x45, 0x45);
        colors[1] = Color.argb(0xff, 0x45, 0xff, 0x45);
        colors[2] = Color.argb(0xff, 0x45, 0x45, 0xff);
        colors[3] = Color.argb(0xff, 0xff, 0xff, 0x45);
        colors[4] = Color.argb(0xff, 0x45, 0xff, 0xff);

        this.stuff = stuff;
    }

    /*
     * Does the actual work of connecting columns in the cursor to a view. Since
     * photos need to be loaded asynchronously via Volley, also kicks off that
     * work via a PhotoAsyncTask.
     */
    private void bindCursorRow(Context context, Cursor c, EventSummaryView esv) {
        String[] columns = c.getColumnNames();
        String colName;
        for (int i = 0; i < columns.length; ++i) {
            colName = columns[i];
            if ("which".equals(colName)) {
                String which = c.getString(i);
                esv.setTitle(which);
                // if we have never seen this particular sensor name before,
                // assign it a color. Maybe one day move this to user prefs?
                if (colorMap.containsKey(which)) {
                    esv.setHighlightColor(colorMap.get(which));
                } else {
                    int color;
                    if (usedColors >= colors.length) {
                        color = defaultColor;
                    } else {
                        color = colors[usedColors];
                        usedColors++;
                        colorMap.put(which, color);
                        context.getSharedPreferences("colormap",
                                Context.MODE_PRIVATE).edit()
                                .putInt(which, color).commit();
                    }
                }
            } else if ("event".equals(colName)) {
                // TODO: this is a kludge. Need to improve the server protocol
                // here
                String type = c.getString(i);
                esv.setAjar("Ajar".equals(type));
                // esv.setOngoing(isOngoing);
            } else if ("ts".equals(colName)) {
                SimpleDateFormat sdf = new SimpleDateFormat(
                        "yyyy'-'MM'-'dd'T'HH:mm:ss", Locale.US);
                try {
                    esv.setTime(sdf.parse(c.getString(i)).getTime());
                } catch (ParseException e) {
                    Log.w("PanopticonCursorAdapter",
                            "exception parsing database timestamp", e);
                }
            } else if ("eventid".equals(colName)) {
                String eventId = c.getString(i);
                (new PhotoAsyncTask(stuff, esv, eventId)).execute();
            }
        }
    }

    @Override
    public void bindView(View view, Context context, Cursor c) {
        EventSummaryView esv = (EventSummaryView) view;
        esv.cancelPendingRequests();
        esv.hideImages();
        bindCursorRow(context, c, esv);
    }

    @Override
    public View newView(Context context, Cursor c, ViewGroup parent) {
        EventSummaryView esv = new EventSummaryView(context);
        return esv;
    }
}
