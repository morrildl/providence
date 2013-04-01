package dude.morrildl.providence;

import java.io.File;
import java.io.IOException;
import java.net.MalformedURLException;
import java.net.URL;
import java.security.KeyManagementException;
import java.security.KeyStore;
import java.security.KeyStoreException;
import java.security.NoSuchAlgorithmException;
import java.security.cert.CertificateException;
import java.util.Properties;

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


public class Stuff {
    private class UrlRewriter implements HurlStack.UrlRewriter {
        @Override
        public String rewriteUrl(String originalUrl) {
            originalUrl = originalUrl.replaceFirst("^http:", "https:");
            try {
                URL parsedUrl = new URL(originalUrl);
                if (!canonicalServerName.equals(parsedUrl.getHost() + ":"
                        + parsedUrl.getPort())) {
                    Log.e("Network.UrlRewriter", "suppressing non-canonical URL " + originalUrl);
                    return null;
                }
            } catch (MalformedURLException e) {
                Log.e("Network.UrlRewriter", "suppressing bogus URL", e);
                return null;
            }

            return originalUrl;
        }
    }
    
    private String canonicalServerName;
    private String photosBase;
    private String vbofSendUrl;
    private String regIdUrl;

    private String oAuthAudience;

    private static Stuff instance = null;

    private static RequestQueue rq = null;

    public static Stuff getInstance(Context context) throws KeyStoreException {
        if (instance == null) {
            synchronized (Stuff.class) {
                if (instance == null) {
                    instance = new Stuff(context);
                }
            }
        }
        if (instance.sslContext == null) {
            throw new KeyStoreException("failure preparing SSL keystore");
        }

        return instance;
    }

    private Context context = null;

    private SSLContext sslContext = null;

    private Stuff(Context context) {
        this.context = context.getApplicationContext();

        try {
            KeyStore ks = KeyStore.getInstance("BKS");
            ks.load(context.getResources().openRawResource(R.raw.keystore),
                    "boogaflex".toCharArray());
            TrustManagerFactory tmf = TrustManagerFactory.getInstance(TrustManagerFactory
                    .getDefaultAlgorithm());
            tmf.init(ks);
            sslContext = SSLContext.getInstance("TLS");
            sslContext.init(null, tmf.getTrustManagers(), null);
        } catch (KeyStoreException e) {
            sslContext = null;
        } catch (KeyManagementException e) {
            sslContext = null;
        } catch (NoSuchAlgorithmException e) {
            sslContext = null;
        } catch (CertificateException e) {
            sslContext = null;
        } catch (NotFoundException e) {
            sslContext = null;
        } catch (IOException e) {
            sslContext = null;
        }

        File cacheDir = context.getDir("photocache", 0);
        DiskBasedCache dbc = new DiskBasedCache(cacheDir, 1024 * 1024 * 10);
        HurlStack stack = new HurlStack(new UrlRewriter(), getSslSocketFactory());
        BasicNetwork bn = new BasicNetwork(stack);
        rq = new RequestQueue(dbc, bn, 4);
        rq.start();
        
        Properties props = new Properties();
        try {
            props.load(context.getResources().openRawResource(
                    R.raw.config));
        } catch (Exception e) {
            Log.e("Config.loadConfig", "exception during load", e);
        }

        oAuthAudience = props.getProperty("OAUTH_AUDIENCE");
        regIdUrl = props.getProperty("REGID_URL");
        vbofSendUrl = props.getProperty("VBOF_SEND_URL");
        photosBase = props.getProperty("PHOTO_BASE");
        canonicalServerName = props.getProperty("CANONICAL_SERVER_NAME");
    }
    public String fetchAuthToken() throws OAuthException {
        try {
            Account[] emails = AccountManager.get(context).getAccountsByType("com.google");
            String email = null;
            for (Account account : emails) {
                if (account.name.endsWith("@gmail.com")) {
                    email = account.name;
                    break;
                }
            }
            if (email == null) {
                throw new OAuthException("couldn't find a Gmail account");
            }
            return GoogleAuthUtil.getToken(context, email, oAuthAudience);
        } catch (IOException e) {
            throw new OAuthException(e);
        } catch (UserRecoverableAuthException e) {
            throw new OAuthException(e);
        } catch (GoogleAuthException e) {
            throw new OAuthException(e);
        }
    }

    public String getCanonicalServerName() {
        return canonicalServerName;
    }

    public String getOAuthAudience() {
        return oAuthAudience;
    }
    public String getPhotosBase() {
        return photosBase;
    }

    public String getRegIdUrl() {
        return regIdUrl;
    }

    public RequestQueue getRequestQueue() {
        return rq;
    }

    public SSLSocketFactory getSslSocketFactory() {
        return sslContext.getSocketFactory();
    }

    public String getVbofSendUrl() {
        return vbofSendUrl;
    }
}
