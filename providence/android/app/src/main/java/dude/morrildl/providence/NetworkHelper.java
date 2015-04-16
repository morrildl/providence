package dude.morrildl.providence;

import java.io.File;
import java.io.IOException;
import java.net.MalformedURLException;
import java.net.URL;
import java.security.KeyManagementException;
import java.security.KeyStore;
import java.security.KeyStoreException;
import java.security.NoSuchAlgorithmException;

import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSocketFactory;
import javax.net.ssl.TrustManagerFactory;

import android.accounts.Account;
import android.accounts.AccountManager;
import android.content.Context;
import android.content.res.Resources.NotFoundException;
import android.util.Log;

import com.android.volley.RequestQueue;
import com.android.volley.toolbox.BasicNetwork;
import com.android.volley.toolbox.DiskBasedCache;
import com.android.volley.toolbox.HurlStack;
import com.google.android.gms.auth.GoogleAuthException;
import com.google.android.gms.auth.GoogleAuthUtil;
import com.google.android.gms.auth.UserRecoverableAuthException;

/**
 * Various wrappers for common operations related to making network requests to
 * the server. Generally these methods wrap calls to Volley doing chores like
 * setting up certificate chains, authentication, and such.
 */
public class NetworkHelper {
    private class UrlRewriter implements HurlStack.UrlRewriter {
        @Override
        public String rewriteUrl(String originalUrl) {
            originalUrl = originalUrl.replaceFirst("^http:", "https:");
            try {
                URL parsedUrl = new URL(originalUrl);
                if (!config.getCanonicalServerName().equals(
                        parsedUrl.getHost() + ":" + parsedUrl.getPort())) {
                    Log.e("Network.UrlRewriter",
                            "suppressing non-canonical URL " + originalUrl);
                    return null;
                }
            } catch (MalformedURLException e) {
                Log.e("Network.UrlRewriter", "suppressing bogus URL", e);
                return null;
            }

            return originalUrl;
        }
    }

    private static NetworkHelper instance = null;

    private static RequestQueue rq = null;

    public static NetworkHelper getInstance(Context context) {
        if (instance == null) {
            synchronized (NetworkHelper.class) {
                if (instance == null) {
                    instance = new NetworkHelper(context);
                }
            }
        }
        if (instance.sslContext == null) {
            Log.e("Stuff.getInstance",
                    "error loading keystore; requests will fail");
        }

        return instance;
    }

    private Context context = null;
    private SSLContext sslContext = null;
    private Config config;

    private NetworkHelper(Context context) {
        this.context = context.getApplicationContext();
        config = Config.getInstance(context);

        try {
            KeyStore ks = config.getKeystore();
            TrustManagerFactory tmf = TrustManagerFactory
                    .getInstance(TrustManagerFactory.getDefaultAlgorithm());
            tmf.init(ks);
            sslContext = SSLContext.getInstance("TLS");
            sslContext.init(null, tmf.getTrustManagers(), null);
        } catch (KeyStoreException e) {
            sslContext = null;
        } catch (KeyManagementException e) {
            sslContext = null;
        } catch (NoSuchAlgorithmException e) {
            sslContext = null;
        } catch (NotFoundException e) {
            sslContext = null;
        }

        File cacheDir = context.getDir("photocache", 0);
        DiskBasedCache dbc = new DiskBasedCache(cacheDir, 1024 * 1024 * 10);
        HurlStack stack = new HurlStack(new UrlRewriter(),
                getSslSocketFactory());
        BasicNetwork bn = new BasicNetwork(stack);
        rq = new RequestQueue(dbc, bn, 4);
        rq.start();
    }

    public String fetchAuthToken() throws OAuthException {
        try {
            Account[] emails = AccountManager.get(context).getAccountsByType(
                    "com.google");
            String email = null;
            // this is somewhat crude: pick the first gmail we see. Should
            // probably include a chooser dialog, but so far no users have > 1
            // @gmail.com
            for (Account account : emails) {
                if (account.name.endsWith("@gmail.com")) {
                    email = account.name;
                    break;
                }
            }
            if (email == null) {
                throw new OAuthException("couldn't find a Gmail account");
            }
            return GoogleAuthUtil.getToken(context, email,
                    config.getOAuthAudience());
        } catch (IOException e) {
            throw new OAuthException(e);
        } catch (UserRecoverableAuthException e) {
            throw new OAuthException(e);
        } catch (GoogleAuthException e) {
            throw new OAuthException(e);
        }
    }

    public RequestQueue getRequestQueue() {
        return rq;
    }

    public SSLSocketFactory getSslSocketFactory() {
        return sslContext.getSocketFactory();
    }

    /*
     * private void storeEvent(Event event) { OpenHelper helper = new
     * OpenHelper(context); SQLiteDatabase db = helper.getWritableDatabase();
     * 
     * // insert all incoming events into the DB, even motion events
     * ContentValues values = new ContentValues(); values.put("which",
     * event.which); values.put("type", event.typeName); values.put("event",
     * event.eventName); values.put("ts", event.timestamp);
     * values.put("eventid", event.id); db.insert("events", null, values); }
     */
}
