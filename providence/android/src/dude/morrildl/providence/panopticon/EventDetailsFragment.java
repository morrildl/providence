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
package dude.morrildl.providence.panopticon;

import java.text.ParseException;
import java.text.SimpleDateFormat;

import android.app.Fragment;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import dude.morrildl.providence.R;

public class EventDetailsFragment extends Fragment {
	private long rowId;

	@Override
	public View onCreateView(LayoutInflater inflater, ViewGroup container,
			Bundle savedInstanceState) {
		return inflater.inflate(R.layout.event_details_fragment, container,
				false);
	}

	@Override
	public void onHiddenChanged(boolean hidden) {
		super.onHiddenChanged(hidden);
		if (!hidden) {
			OpenHelper helper = new OpenHelper(this.getActivity());
			SQLiteDatabase db = helper.getReadableDatabase();
			Cursor c = db.query("events", new String[] { "which", "type",
					"event", "ts" }, "rowid = ?",
					new String[] { Long.toString(rowId) }, null, null, null);

			StringBuilder sb = new StringBuilder();
			if (c.moveToFirst()) {
				sb.append(c.getString(0));
				sb.append(" reported '").append(c.getString(2)).append("' ");

				try {
					SimpleDateFormat sdf = new SimpleDateFormat(
							"yyyy'-'MM'-'dd'T'HH:mm:ss");
					java.util.Date parsedTs = sdf.parse(c.getString(3));

					sdf = new SimpleDateFormat(
							"EEE, dd MMMMMMMMM yyyy 'at' KK:mm:ssa");
					sb.append("on ").append(sdf.format(parsedTs));
				} catch (ParseException e) {
				}
				((TextView) getActivity().findViewById(
						R.id.event_details_fragment_main_text)).setText(sb
						.toString());
			} else {
				((TextView) getActivity().findViewById(
						R.id.event_details_fragment_main_text))
						.setText(R.string.latest_event_default_text);
			}
		}
	}

	public void setId(long id) {
		this.rowId = id;
	}
}
