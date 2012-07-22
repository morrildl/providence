package dude.morrildl.lifer.providence;

import android.app.Fragment;
import android.os.Bundle;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import dude.morrildl.lifer.R;

public class LatestEventFragment extends Fragment {
	@Override
	public View onCreateView(LayoutInflater inflater, ViewGroup container,
			Bundle savedInstanceState) {
		return inflater.inflate(R.layout.latest_event_fragment, container,
				false);
	}

	@Override
	public void onResume() {
		super.onResume();
		Bundle b = getActivity().getIntent().getExtras();
		if (b != null) {
			StringBuilder sb = new StringBuilder();
			sb.append(b.getString("WhichName"));
			String s = b.getString("SensorTypeName");
			if (s != null && !"".equals(s)) {
				sb.append(" (a ").append(s).append(")");
			}
			sb.append(" reported '").append(b.getString("EventName"))
					.append("' at ").append(b.getString("When"));
			((TextView) getActivity().findViewById(
					R.id.latest_event_fragment_main_text)).setText(sb
					.toString());
		} else {
			Log.e("booga booga wtf", "no extras");
		}
	}
}
