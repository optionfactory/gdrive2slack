(function() {
    var developerKey = 'AIzaSyA41Ht18bKcPh6wM2-RfHZubxBfs1Oox0g';
    var clientId = "906425490329-ek2j1r0a58k9skb9lq5micpfhpeihqlp.apps.googleusercontent.com";
    var scope = ['https://www.googleapis.com/auth/drive.readonly'];
    var clientLoaded = false;

    window.DrivePicker = function(cb) {
        this.oauthToken = null;
        this.driveLoaded = false;
        this.rootFolderId = null;
        this.pickerLoaded = false;
        this.cb = cb;

        var self = this;
    }

    window.DrivePicker.prototype = {
        _loadRootFolder: function() {
            var self = this;
            if(this.oauthToken && this.driveLoaded) {
                gapi.client.drive.about.get().execute(function(resp) {
                    self.rootFolderId = resp.rootFolderId;
                    self._pick();
                })
            }
        },
        _pick: function() {
            if(this.oauthToken && this.rootFolderId && this.pickerLoaded) {
                this.createPicker();
            }
        },
        pick: function() {
            var self = this;
            
            gapi.client.load("drive", "v2", function() {
                self.driveLoaded = true;
                self._loadRootFolder();
            });
            gapi.load('picker', {'callback': function() {
                self.pickerLoaded = true;
                self._pick();
            }});
    
            gapi.auth.init(function() {
                window.gapi.auth.authorize({
                    'client_id': clientId,
                    'scope': scope,
                    'immediate': false
                }, function(authResult) {
                    if (authResult && !authResult.error) {
                        self.oauthToken = authResult.access_token;
                        self._loadRootFolder();
                    }
                });
            });
        },
        createPicker: function() {
            var view = new google.picker.DocsView();
            view.setMimeTypes("application/vnd.google-apps.folder");
            view.setIncludeFolders(true);
            view.setSelectFolderEnabled(true);
            view.setParent(this.rootFolderId);
            var picker = new google.picker.PickerBuilder().
                addView(view).
                setOAuthToken(this.oauthToken).
                setDeveloperKey(developerKey).
                setCallback(this.pickerCallback.bind(this)).
                build();
            picker.setVisible(true);
        },
        pickerCallback: function(data) {
            if (data[google.picker.Response.ACTION] == google.picker.Action.PICKED) {
                var doc = data[google.picker.Response.DOCUMENTS][0];
                this.cb(doc);
            }
        }
    };
})();