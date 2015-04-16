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

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Locale;

import android.content.Context;
import android.graphics.Bitmap;
import android.graphics.Bitmap.Config;
import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Paint;
import android.text.Layout;
import android.text.StaticLayout;
import android.text.TextPaint;
import android.util.AttributeSet;
import android.util.DisplayMetrics;
import android.util.Log;
import android.view.View;
import android.view.ViewGroup;
import android.view.WindowManager;
import android.widget.ImageView;

import com.android.volley.Request;
import com.android.volley.Response.ErrorListener;
import com.android.volley.Response.Listener;
import com.android.volley.VolleyError;

import dude.morrildl.providence.R;
import dude.morrildl.providence.NetworkHelper;
import dude.morrildl.providence.volley.ImageRequest;

public class EventSummaryView extends ViewGroup {
    protected static final String TAG = "EventSummaryView";
    private final int badgeHeight;
    private String badgeText = null;
    private final int badgeWidth;
    private final int eightDips;
    private int imageCount = 0;
    private int imageIndentLeft = 0;
    private int imageIndentTop = 0;
    private float imageRatio = 1.666667f;
    private final ImageView[] images = new ImageView[4];
    private boolean isAjar = false;
    private boolean isOngoing = false;
    private int measuredHeight;
    private int measuredWidth;
    private final DisplayMetrics metrics;
    private final TextPaint paint = new TextPaint();
    private int pedestalColor = 0;
    private long time = 0;
    private String timeText = null;
    private String title = null;
    private ArrayList<Request<?>> volleyRequests = null;

    public EventSummaryView(Context context) {
        this(context, null, 0);
    }

    public EventSummaryView(Context context, AttributeSet attrs, int defStyle) {
        super(context, attrs, defStyle);

        // 8dip is a standard multiple for much of the UI rhythm; precompute it
        metrics = new DisplayMetrics();
        ((WindowManager) getContext().getSystemService(Context.WINDOW_SERVICE))
                .getDefaultDisplay().getMetrics(metrics);
        eightDips = Math.round(metrics.density * 8);

        // compute the offsets by which the tiled images are "indented" behind
        // the badge
        imageIndentTop = 3 * eightDips;
        imageIndentLeft = 3 * eightDips;

        // compute the size of the badge. Note that this is unaffected by
        // padding etc.
        // as it is part of the internal layout and is proportional (scaled) per
        // density
        badgeWidth = eightDips * 7 * 3;
        badgeHeight = eightDips * 5 * 2;

        // setup images to a sane initial state
        for (int i = 0; i < 4; ++i) {
            images[i] = new ImageView(context);
            addView(images[i]);
        }
        imageCount = 1;
        images[0].setVisibility(View.VISIBLE);
        images[0].setImageResource(R.drawable.im_default_camera);
        for (int i = 1; i < 4; ++i) {
            images[i].setVisibility(View.GONE);
        }
        for (int i = 0; i < 4; ++i) {
            images[i].setAdjustViewBounds(false);
            images[i].setPadding(0, 0, 0, 0);
            images[i].setScaleType(ImageView.ScaleType.CENTER_CROP);

        }

        // set a default for color-coding #HOLOYOLO
        pedestalColor = context.getResources().getColor(
                android.R.color.holo_blue_bright);

        updateBadgeText();

        setPadding(eightDips, eightDips / 2, eightDips, 0);
    }

    /**
     * Clears any outstanding Volley requests. This is important to avoid race
     * conditions: since things like a ListView will recycle views, it's very
     * common to have a given EventSummaryInstance get recycled while it still
     * has pending requests for images. This tells the View to cancel any such
     * pending requests so that they don't come in after legit requests and
     * clobber them.
     */
    public void cancelPendingRequests() {
        if (volleyRequests != null) {
            for (Request<?> request : volleyRequests) {
                request.cancel();
            }
        }
        volleyRequests = null;
    }

    @Override
    protected void dispatchDraw(Canvas canvas) {
        // let the framework draw the tiled images first, as they are set up as
        // plain old children of this View
        super.dispatchDraw(canvas);

        // First we're going to draw some boxes with sharp corners, so
        // anti-aliasing is unnecessary
        paint.setStyle(Paint.Style.FILL_AND_STROKE);
        paint.setAntiAlias(false);

        // draw a drop-shadow for the floating badge mini-card. Note that it has
        // to leave room for both the badge box and the pedestal underline box
        paint.setColor(Color.argb(0x88, 0, 0, 0));
        canvas.drawRect(eightDips / 2, eightDips / 2, badgeWidth + eightDips
                / 2, badgeHeight + eightDips + eightDips / 2, paint);

        // draw the main badge floating mini-card rect, and its pedestal
        paint.setColor(getResources().getColor(R.color.badge_bg));
        canvas.drawRect(0, 0, badgeWidth, badgeHeight, paint);
        paint.setColor(pedestalColor);
        canvas.drawRect(0, badgeHeight, badgeWidth, badgeHeight + eightDips,
                paint);

        // now draw a border around the floating card, as background protection
        paint.setColor(Color.argb(0x88, 0, 0, 0));
        paint.setStyle(Paint.Style.STROKE);
        paint.setStrokeWidth(eightDips / 4);
        canvas.drawRect(0, 0, badgeWidth, badgeHeight + eightDips, paint);
        paint.setStrokeWidth(0f);

        // boxes are done, now set up the paint for drawing text
        paint.setStyle(Paint.Style.FILL_AND_STROKE);
        paint.setColor(getResources().getColor(R.color.badge_text));
        paint.setAntiAlias(true);

        // draw the main title text
        paint.setTextSize(40);
        int pleft = getPaddingLeft();
        int pright = getPaddingRight();
        int ptop = getPaddingTop();
        int pbottom = getPaddingBottom();
        StaticLayout sl = new StaticLayout(badgeText, paint, badgeWidth - pleft
                - pright, Layout.Alignment.ALIGN_NORMAL, 1f, 0f, false);
        canvas.translate(pleft, ptop);
        sl.draw(canvas);

        // draw the date & time
        paint.setTextSize(30);
        sl = new StaticLayout(timeText, paint, badgeWidth - pleft - pright,
                Layout.Alignment.ALIGN_NORMAL, 1f, 0f, false);
        canvas.translate(0, badgeHeight - sl.getHeight() - ptop - pbottom);
        sl.draw(canvas);
    }

    /*
     * Used to make all 4 images invisible in the layout. Without this, if the
     * view gets recycled, its images will be displaying the bitmaps from the
     * previous incarnation. These will then update very quickly as Volley
     * requests come back, making the UI look flickery and sloppy. This hides
     * the bitmaps, so that when the view scrolls into the viewport, it is blank
     * until it fills in with proper images.
     */
    public void hideImages() {
        for (int i = 0; i < 4; ++i) {
            images[i].setVisibility(View.INVISIBLE);
        }
    }

    /*
     * Alternative to calling setImageCount() and setBitmap(). When called, the
     * view uses Volley to fetch the indicated URLs, and stuff them (in order)
     * into its image slots.
     */
    public void loadImages(String token, ArrayList<String> urls) {
        cancelPendingRequests();

        if (urls.size() < 1) {
            setImageCount(0);
            return;
        }

        setImageCount(urls.size());
        volleyRequests = new ArrayList<Request<?>>();

        for (int i = 0; i < urls.size() && i < 4; ++i) {
            final String url = urls.get(i).trim();
            final int imageIndex = i;

            ImageRequest ir = new ImageRequest(url, new Listener<Bitmap>() {
                @Override
                public void onResponse(Bitmap response) {
                    setBitmap(imageIndex, response);
                }
            }, 0, 0, Config.ARGB_4444, new ErrorListener() {
                @Override
                public void onErrorResponse(VolleyError error) {
                    String code = error.networkResponse != null ? ""
                            + error.networkResponse.statusCode : "";
                    Log.w(TAG, "volley responded with error " + code,
                            error.getCause());
                }
            });
            ir.setHeader("X-OAuth-JWT", token);
            ir.setShouldCache(true);
            NetworkHelper.getInstance(getContext()).getRequestQueue().add(ir);
            volleyRequests.add(ir);
        }
    }

    @Override
    protected void onLayout(boolean changed, int l, int t, int r, int b) {
        // This View is a tiled set of 4 images, overlaid with a floating badge
        // containing general info about the event. The images are "indented" at
        // the top and bottom (i.e. are not full-bleed on those edges) to help
        // with the "floating" illusion of the badge. The tiled images ARE
        // full-bleed on bottom and right.
        int imgWidth = ((Math.abs(r - l) - imageIndentLeft) / 2);
        int imgHeight = ((Math.abs(b - t) - imageIndentTop) / 2);
        if (imageCount < 2) {
            images[0].layout(imageIndentLeft, imageIndentTop, imgWidth * 2,
                    imgHeight * 2);
        } else {
            for (int i = 0; i < imageCount; ++i) {
                int left = imageIndentLeft + ((i % 2 == 0) ? 0 : imgWidth);
                int right = left + imgWidth;

                int top = imageIndentTop + ((i / 2 == 0) ? 0 : imgHeight);
                int bottom = top + imgHeight;

                images[i].layout(left, top, right, bottom);
            }
        }
    }

    @Override
    protected void onMeasure(int widthMeasureSpec, int heightMeasureSpec) {
        // we always take up all the width the parent is willing to give us, and
        // as much height as we need to fit our images
        measuredWidth = View.MeasureSpec.getSize(widthMeasureSpec);
        int widthSpec = View.MeasureSpec.makeMeasureSpec(measuredWidth,
                View.MeasureSpec.EXACTLY);

        measuredHeight = Math.round(measuredWidth / imageRatio);
        if (measuredHeight % 2 != 0) {
            measuredHeight += 1;
        }
        int heightSpec = View.MeasureSpec.makeMeasureSpec(measuredHeight,
                View.MeasureSpec.EXACTLY);

        setMeasuredDimension(widthSpec, heightSpec);
    }

    public void setAjar(boolean isAjar) {
        this.isAjar = isAjar;
        updateBadgeText();
    }

    /**
     * Sets a Bitmap for a particular slot. This also sets the bitmap to
     * View.VISIBLE so that it appears in the UI.
     */
    public void setBitmap(int which, Bitmap bitmap) {
        if (which < 0 || which >= imageCount) {
            throw new IllegalArgumentException("bitmap index out of bounds");
        }
        images[which].setImageBitmap(bitmap);
        images[which].setVisibility(View.VISIBLE);
        imageRatio = ((float) bitmap.getWidth()) / bitmap.getHeight();
        imageRatio = (imageRatio < 1) ? 1 / imageRatio : imageRatio;
    }

    /**
     * Sets the color of the pedestal underlining the floating badge box,
     * intended to be used to color-code like entries.
     */
    public void setHighlightColor(int color) {
        pedestalColor = color;
    }

    /**
     * Since images can come in asynchronously via calls to setBitmap(), this
     * method tells the class how many to expect.
     */
    public void setImageCount(int imageCount) {
        // if we don't have any images, reset the 0th image to the default, and
        // remove all others from UI
        if (imageCount < 1) {
            this.imageCount = 1;
            images[0].setImageResource(R.drawable.im_default_camera);
            images[0].setVisibility(View.VISIBLE);
            for (int i = 1; i < 4; ++i) {
                images[i].setVisibility(View.GONE);
            }
            return;
        }

        this.imageCount = imageCount;
        int i;
        for (i = 0; i < imageCount && i < 4; ++i) {
            images[i].setVisibility(View.INVISIBLE);
        }
        for (int j = i; j < 4; ++j) {
            images[j].setVisibility(View.GONE);
        }
    }

    public void setOngoing(boolean isOngoing) {
        this.isOngoing = isOngoing;
        updateBadgeText();
    }

    public void setTime(long time) {
        if (time < 0) {
            return;
        }
        this.time = time;
        updateBadgeText();
    }

    public void setTitle(String title) {
        this.title = title;
        updateBadgeText();
    }

    /*
     * Because we're a ViewGroup...
     */
    @Override
    public boolean shouldDelayChildPressedState() {
        return false;
    }

    /**
     * To avoid having to do string inspection and concatenation in the
     * rendering loop, this method pre-processes key strings whenever their
     * constituent parts are updated.
     */
    private void updateBadgeText() {
        StringBuffer sb = new StringBuffer();
        if (title != null) {
            sb.append(title).append("\n");
        }
        if (isAjar) {
            sb.append("AJAR").append("\n");
        } else if (isOngoing) {
            sb.append("Ongoing").append("\n");
        }
        badgeText = sb.toString();

        sb = new StringBuffer();
        if (time > 0) {
            SimpleDateFormat sdf = new SimpleDateFormat("h:mm:ssa EEE dd MMM",
                    Locale.US);
            sb.append(sdf.format(time));
        }
        timeText = sb.toString();
    }
}
