var app = new Vue({
    el: "#settings-app",
    data: {
        currentTab: "site",
    },
    mounted: function() {
        var self = this;

        var m = window.location.hash.match(/^#(\w+?)$/)
        if (m) {
            self.currentTab = m[1];
        }
    },
    methods: {
        load: function() {

        },
    }
})
